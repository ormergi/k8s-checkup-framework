package latencycheck

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"k8s.io/utils/net"
	v1 "kubevirt.io/api/core/v1"
	"kubevirt.io/client-go/kubecli"

	expect "github.com/google/goexpect"

	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/config"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/console"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/ping"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/preflight"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/utils/nads"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/utils/vmis"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/utils/vmis/libvmi"
)

const (
	networkTypeSRIOV     = "sriov"
	networkTypeBridge    = "bridge"
	networkTypeCNVBridge = "cnv-bridge"
)

const (
	mac1 = "02:00:00:f9:32:1f"
	mac2 = "02:00:00:7b:55:76"

	cidr1 = "192.168.0.100/24"
	cidr2 = "192.168.0.200/24"
)

const defaultLatencySampleDuration = time.Second * 5

const namespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

type options struct {
	workingNamespace  string
	networkNamespace  string
	networkName       string
	sourceNode        string
	targetNode        string
	sampleDuration    time.Duration
	desiredMaxLatency time.Duration
	sourceMacAddr     string
	targetMacAddr     string
	sourceCIDR        string
	targetCIDR        string
}

func CreateLatencyCheckOptions(params map[string]string) (options, error) {
	const errMsgPrefix = "failed to create latency check options"

	const (
		secondsUnit      = "s"
		millisecondsUnit = "ms"
	)

	var err error
	var sampleDuration time.Duration
	sampleDurationEnvVarValue := params[config.SampleDurationSecondsEnvVarName]
	if sampleDurationEnvVarValue == "" {
		sampleDuration = defaultLatencySampleDuration
	} else {
		sampleDuration, err = time.ParseDuration(sampleDurationEnvVarValue + secondsUnit)
		if err != nil {
			return options{}, fmt.Errorf("%s: failed to parse env var %s: %v",
				errMsgPrefix, config.SampleDurationSecondsEnvVarName, err)
		}
	}

	var desiredMaxLatency time.Duration
	desiredMaxLatencyEnvVarValue := params[config.DesiredMaxLatencyMillisecondsEnvVarName]
	if desiredMaxLatencyEnvVarValue == "" {
		desiredMaxLatency = 0
	} else {
		desiredMaxLatency, err = time.ParseDuration(desiredMaxLatencyEnvVarValue + millisecondsUnit)
		if err != nil {
			return options{}, fmt.Errorf("%s: failed to parse env var %s: %v",
				errMsgPrefix, config.DesiredMaxLatencyMillisecondsEnvVarName, err)
		}
	}

	workingNamespace, err := os.ReadFile(namespaceFile)
	if err != nil {
		return options{}, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	options := options{
		workingNamespace:  string(workingNamespace),
		networkNamespace:  params[config.NetworkNamespaceEnvVarName],
		networkName:       params[config.NetworkNameEnvVarName],
		sourceNode:        params[config.SourceNodeNameEnvVarName],
		targetNode:        params[config.TargetNodeNameEnvVarName],
		sampleDuration:    sampleDuration,
		desiredMaxLatency: desiredMaxLatency,
		sourceMacAddr:     mac1,
		sourceCIDR:        cidr1,
		targetMacAddr:     mac2,
		targetCIDR:        cidr2,
	}

	return options, nil
}

func StartLatencyCheck(virtClient kubecli.KubevirtClient, options options) (ping.Result, error) {
	if err := runNetworkLatencyPreflightChecks(virtClient, options); err != nil {
		return ping.Result{}, err
	}

	sourceVMI, targetVMI, err := startNetworkLatencyCheckVMIs(virtClient, options)
	if err != nil {
		return ping.Result{}, err
	}
	defer func() {
		if err := vmis.DeleteAndWaitVmisDispose(virtClient, sourceVMI, targetVMI); err != nil {
			log.Println(err.Error())
		}
	}()

	result, err := runNetworkLatencyCheck(virtClient, options.networkName, sourceVMI, targetVMI, options.sampleDuration)
	if err != nil {
		return result, err
	}

	if options.desiredMaxLatency > 0 &&
		result.Max() > options.desiredMaxLatency {
		return result, fmt.Errorf("max latency is greater than expected:\n\texpected: (%v)\n\tresult: (%v)",
			options.desiredMaxLatency, result.Max())
	}

	return result, nil
}

func runNetworkLatencyPreflightChecks(virtClient kubecli.KubevirtClient, options options) error {
	const errMsgPrefix = "not all preflight checks passed"

	log.Println("Starting preflights checks..")

	if err := preflight.VerifyKubevirtAvailable(virtClient); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	if err := preflight.VerifyNetworkAttachmentDefinitionExists(virtClient, options.networkNamespace, options.networkName); err != nil {
		return fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	return nil
}

func runNetworkLatencyCheck(virtClient kubecli.KubevirtClient, netwrorkName string, sourceVMI, targetVMI *v1.VirtualMachineInstance, duration time.Duration) (ping.Result, error) {
	const errMsgPrefix = "network latency check failed"

	targetVmiIP, err := vmis.GetVmiNetwrokIPAddress(virtClient, targetVMI.Namespace, targetVMI.Name, netwrorkName)
	if err != nil {
		return ping.Result{}, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	pingStart := time.Now()
	responses, err := pingFromVMConsole(duration, sourceVMI, targetVmiIP)
	if err != nil {
		return ping.Result{}, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}
	pingDuration := time.Now().Sub(pingStart)

	pingResult := ping.ParsePingResult(responses)

	pingResult.SetMeasurementDuration(pingDuration)

	return pingResult, nil
}

func startNetworkLatencyCheckVMIs(virtClient kubecli.KubevirtClient, options options) (*v1.VirtualMachineInstance, *v1.VirtualMachineInstance, error) {
	const errMsgPrefix = "failed to setup netwrok latency check"

	var fn createLatencyCheckVmiFn

	nad, err := nads.Get(virtClient, options.networkNamespace, options.networkName)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}
	cnis, err := nads.ConfigCniPlugins(nad)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}
	if len(cnis) < 1 {
		return nil, nil, fmt.Errorf("%s: no CNI pluging found at the NetwrokAttachmentDefinition config", errMsgPrefix)
	}
	cni := cnis[0]
	if !isSupportedCNIPlugin(cni) {
		return nil, nil, fmt.Errorf("%s: the NetwrokAttachmentDefinition uses unsupported CNI, try %v",
			errMsgPrefix, supportedCNIPlugins)
	}

	switch cni {
	case networkTypeBridge, networkTypeCNVBridge:
		fn = newLatencyCheckVmiWithBridgeIface
	case networkTypeSRIOV:
		fn = newLatencyCheckVmiWithSriovIface
	}
	sourceVMI := fn(options.workingNamespace, options.networkNamespace, options.networkName, options.sourceMacAddr, options.sourceCIDR, options.sourceNode)
	targetVMI := fn(options.workingNamespace, options.networkNamespace, options.networkName, options.targetMacAddr, options.targetCIDR, options.targetNode)

	if err := vmis.StartAndWaitVmisReady(virtClient, sourceVMI, targetVMI); err != nil {
		return nil, nil, err
	}

	ipacmd := "ip a"
	_, err = console.RunCommand(sourceVMI, ipacmd, time.Second*15)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}
	_, err = console.RunCommand(targetVMI, ipacmd, time.Second*15)
	if err != nil {
		return nil, nil, fmt.Errorf("%s: %v", errMsgPrefix, err)
	}

	return sourceVMI, targetVMI, nil
}

var supportedCNIPlugins = []string{"bridge", "bridge-cnv", "sriov"}

func isSupportedCNIPlugin(cni string) bool {
	for _, supportedPlugin := range supportedCNIPlugins {
		if supportedPlugin == cni {
			return true
		}
	}
	return false
}

type createLatencyCheckVmiFn func(namespace, networkNamespace, networkName, mac, cidr, node string) *v1.VirtualMachineInstance

func newLatencyCheckVmiWithBridgeIface(namespace, networkNamespace, networkName, mac, cidr, node string) *v1.VirtualMachineInstance {
	iface := libvmi.InterfaceDeviceWithBridgeBinding(networkName)
	return newLatencyCheckVmi(namespace, networkName, mac, cidr, node,
		libvmi.WithInterface(iface),
		libvmi.WithNetwork(libvmi.MultusNetwork(networkName, fmt.Sprintf("%s/%s", networkNamespace, networkName))),
	)
}

func newLatencyCheckVmiWithSriovIface(namespace, networkNamespace, networkName, mac, cidr, node string) *v1.VirtualMachineInstance {
	iface := libvmi.InterfaceDeviceWithSRIOVBinding(networkName)
	iface.MacAddress = mac
	cloudInitNetworkData, _ := libvmi.NewNetworkData(
		libvmi.WithEthernet("sriovnet",
			libvmi.WithAddresses(cidr),
			libvmi.WithMatchingMAC(mac),
		),
	)
	return newLatencyCheckVmi(namespace, networkName, mac, cidr, node,
		libvmi.WithInterface(iface),
		libvmi.WithNetwork(libvmi.MultusNetwork(networkName, fmt.Sprintf("%s/%s", networkNamespace, networkName))),
		libvmi.WithCloudInitNoCloudNetworkData(cloudInitNetworkData, false),
	)
}

func newLatencyCheckVmi(namespace, networkName, mac, cidr, nodeName string, opts ...libvmi.Option) *v1.VirtualMachineInstance {
	withVmiOptions := []libvmi.Option{
		libvmi.WithNamespace(namespace),
		libvmi.WithNodeSelector(nodeName),
	}
	withVmiOptions = append(withVmiOptions, opts...)
	return libvmi.NewFedora(withVmiOptions...)
}

func pingFromVMConsole(timeout time.Duration, vmi *v1.VirtualMachineInstance, ipAddr string, args ...string) ([]expect.BatchRes, error) {
	cmd := composePingCommand(ipAddr, fmt.Sprintf("-w %d", int(timeout.Seconds())))
	resp, err := console.RunCommand(vmi, cmd, timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to ping from VMI %s/%s to %s, error: %v",
			vmi.Namespace, vmi.Name, ipAddr, err)
	}
	return resp, nil
}

func composePingCommand(ipAddr string, args ...string) string {
	const (
		ping  = "ping"
		ping6 = "ping -6"
	)

	pingString := ping
	if net.IsIPv6String(ipAddr) {
		pingString = ping6
	}

	if len(args) == 0 {
		args = []string{"-c 5 -w 10"}
	}
	args = append([]string{pingString, ipAddr}, args...)

	return strings.Join(args, " ")
}
