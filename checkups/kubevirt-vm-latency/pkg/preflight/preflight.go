package preflight

import (
	"fmt"
	"log"

	k8scorev1 "k8s.io/api/core/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/utils/configmaps"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/utils/nads"
)

func VerifyConfigMapExists(virtClient kubecli.KubevirtClient, namespace, name string) error {
	log.Printf("verifying ConfigMap %s/%s exists..", namespace, name)

	if _, err := configmaps.Get(virtClient, namespace, name); err != nil {
		return err
	}

	return nil
}

func VerifyKubevirtAvailable(virtClient kubecli.KubevirtClient) error {
	const errMsgPrefix = "failed to verify Kubevirt exists and ready: "

	log.Println("verifying Kubevirt deployed and ready..")

	kubevirts, err := virtClient.KubeVirt(k8smetav1.NamespaceAll).List(&k8smetav1.ListOptions{})
	if err != nil {
		return err
	}

	if len(kubevirts.Items) == 0 {
		return fmt.Errorf("%s, could not detect a kubevirt installation", errMsgPrefix)
	}

	if len(kubevirts.Items) > 1 {
		return fmt.Errorf("%s, invalid Kubevirt installation, more than one KubeVirt resource found", errMsgPrefix)
	}

	kubevirt := kubevirts.Items[0]
	for _, condition := range kubevirt.Status.Conditions {
		if condition.Type == v1.KubeVirtConditionAvailable && condition.Status != k8scorev1.ConditionTrue {
			return fmt.Errorf("%s, Kubevirt is not ready", errMsgPrefix)
		}
	}

	return nil
}

func VerifyNetworkAttachmentDefinitionExists(virtClient kubecli.KubevirtClient, namespace, name string) error {
	log.Printf("verifying NetwrokAttachmentDefinition %s/%s exists..", namespace, name)

	if _, err := nads.Get(virtClient, namespace, name); err != nil {
		return err
	}

	return nil
}
