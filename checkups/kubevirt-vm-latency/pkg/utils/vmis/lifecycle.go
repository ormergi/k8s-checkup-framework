package vmis

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	k8scorev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8swait "k8s.io/apimachinery/pkg/util/wait"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	expect "github.com/google/goexpect"
	"google.golang.org/grpc/codes"

	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/console"
)

// LoginToFedora performs a console login to a Fedora base VM
func LoginToFedora(vmi *v1.VirtualMachineInstance) error {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		panic(err)
	}

	expecter, _, err := console.NewExpecter(virtClient, vmi, 10*time.Second)
	if err != nil {
		return err
	}
	defer expecter.Close()

	err = expecter.Send("\n")
	if err != nil {
		return err
	}

	// Do not login, if we already logged in
	b := append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BExp{R: fmt.Sprintf(`(\[fedora@(localhost|%s) ~\]\$ |\[root@(localhost|%s) fedora\]\# )`, vmi.Name, vmi.Name)},
	})
	_, err = expecter.ExpectBatch(b, 5*time.Second)
	if err == nil {
		return nil
	}

	b = append([]expect.Batcher{
		&expect.BSnd{S: "\n"},
		&expect.BSnd{S: "\n"},
		&expect.BCas{C: []expect.Caser{
			&expect.Case{
				// Using only "login: " would match things like "Last failed login: Tue Jun  9 22:25:30 UTC 2020 on ttyS0"
				// and in case the VM's did not get hostname form DHCP server try the default hostname
				R:  regexp.MustCompile(fmt.Sprintf(`(localhost|%s) login: `, vmi.Name)),
				S:  "fedora\n",
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Password:`),
				S:  "fedora\n",
				T:  expect.Next(),
				Rt: 10,
			},
			&expect.Case{
				R:  regexp.MustCompile(`Login incorrect`),
				T:  expect.LogContinue("Failed to log in", expect.NewStatus(codes.PermissionDenied, "login failed")),
				Rt: 10,
			},
			&expect.Case{
				R: regexp.MustCompile(fmt.Sprintf(`\[fedora@(localhost|%s) ~\]\$ `, vmi.Name)),
				T: expect.OK(),
			},
		}},
		&expect.BSnd{S: "sudo su\n"},
		&expect.BExp{R: console.PromptExpression},
	})
	res, err := expecter.ExpectBatch(b, 2*time.Minute)
	if err != nil {
		log.Printf("Login attempt failed: %+v", res)
		// Try once more since sometimes the login prompt is ripped apart by asynchronous daemon updates
		res, err := expecter.ExpectBatch(b, 1*time.Minute)
		if err != nil {
			log.Printf("Retried login attempt after two minutes failed: %+v", res)
			return err
		}
	}

	err = configureConsole(expecter, false)
	if err != nil {
		return err
	}
	return nil
}

func configureConsole(expecter expect.Expecter, shouldSudo bool) error {
	sudoString := ""
	if shouldSudo {
		sudoString = "sudo "
	}
	batch := append([]expect.Batcher{
		&expect.BSnd{S: "stty cols 500 rows 500\n"},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue("0")},
		&expect.BSnd{S: fmt.Sprintf("%sdmesg -n 1\n", sudoString)},
		&expect.BExp{R: console.PromptExpression},
		&expect.BSnd{S: "echo $?\n"},
		&expect.BExp{R: console.RetValue("0")}})
	resp, err := expecter.ExpectBatch(batch, 30*time.Second)
	if err != nil {
		log.Printf("%v\n", resp)
	}
	return err
}

func StartAndWaitVmisReady(virtClient kubecli.KubevirtClient, vmis ...*v1.VirtualMachineInstance) error {
	for _, vmi := range vmis {
		if err := startVmi(virtClient, vmi); err != nil {
			return err
		}
	}
	for _, vmi := range vmis {
		if err := waitVmiReady(virtClient, vmi); err != nil {
			return err
		}
	}

	return nil
}

func startVmi(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) error {
	log.Printf("starting VMI %s/%s..", vmi.Namespace, vmi.Name)
	if _, err := virtClient.VirtualMachineInstance(vmi.Namespace).Create(vmi); err != nil {
		return fmt.Errorf("failed to start VMI %s/%s: %v", vmi.Namespace, vmi.Name, err)
	}
	return nil
}

func waitVmiReady(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) error {
	log.Printf("waiting for VMI %s/%s to be ready..\n", vmi.Namespace, vmi.Name)
	if err := waitForVmiCondition(virtClient, vmi, v1.VirtualMachineInstanceAgentConnected); err != nil {
		return fmt.Errorf("VMI %s/%s was not ready on time: %v", vmi.Namespace, vmi.Name, err)
	}
	if err := LoginToFedora(vmi); err != nil {
		return fmt.Errorf("failed to login to VMI %s/%s: %v", vmi.Namespace, vmi.Name, err)
	}
	return nil
}

func waitForVmiCondition(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance, conditionType v1.VirtualMachineInstanceConditionType) error {
	err := k8swait.PollImmediate(time.Second*1, time.Minute*5, func() (bool, error) {
		updatedVmi, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &k8smetav1.GetOptions{})
		if err != nil {
			return false, nil
		}
		for _, condition := range updatedVmi.Status.Conditions {
			if condition.Type == conditionType && condition.Status == k8scorev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for VMI %s condition %s: %v", vmi.Name, conditionType, err)
	}

	return nil
}

func GetVmiNetwrokIPAddress(virtClient kubecli.KubevirtClient, vmiNamesapce, vmiName, networkName string) (string, error) {
	vmi, err := virtClient.VirtualMachineInstance(vmiNamesapce).Get(vmiName, &k8smetav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get the VMI network IP address: %v", err)
	}

	var ip string
	for _, net := range vmi.Status.Interfaces {
		if net.Name == networkName {
			if net.IP == "" {
				return "", fmt.Errorf("failed to get VMI %s/%s network name %s IP address", vmiNamesapce, vmiName, networkName)
			}
			ip = net.IP
			break
		}
	}

	return ip, nil
}

func DeleteAndWaitVmisDispose(virtClient kubecli.KubevirtClient, vmis ...*v1.VirtualMachineInstance) error {
	var errs []error
	for _, vmi := range vmis {
		if err := deleteVmi(virtClient, vmi); err != nil {
			errs = append(errs, err)
		}
	}
	for _, vmi := range vmis {
		if err := waitForVmiDispose(virtClient, vmi); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return buildErrorMessage(errs)
	}

	return nil
}

func deleteVmi(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) error {
	log.Printf("deleting VMI %s/%s..\n", vmi.Namespace, vmi.Name)
	if err := virtClient.VirtualMachineInstance(vmi.Namespace).Delete(vmi.Name, &k8smetav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("failed to delete VMI %s/%s: %v", vmi.Namespace, vmi.Name, err)
	}
	return nil
}

func waitForVmiDispose(virtClient kubecli.KubevirtClient, vmi *v1.VirtualMachineInstance) error {
	log.Printf("waiting for VMI %s/%s to dispose..\n", vmi.Namespace, vmi.Name)
	err := k8swait.PollImmediate(time.Second*1, time.Minute*5, func() (bool, error) {
		_, err := virtClient.VirtualMachineInstance(vmi.Namespace).Get(vmi.Name, &k8smetav1.GetOptions{})
		if k8serrors.IsNotFound(err) {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("failed to wait for VMI %s/%s to dispose: %v",
			vmi.Name, vmi.Namespace, err)
	}

	return nil
}

func buildErrorMessage(errors []error) error {
	errorMessageBuilder := strings.Builder{}
	for _, err := range errors {
		errorMessageBuilder.WriteString(err.Error() + "\n")
	}
	return fmt.Errorf(errorMessageBuilder.String())
}
