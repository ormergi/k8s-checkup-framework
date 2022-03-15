package main

import (
	"log"

	"kubevirt.io/client-go/kubecli"

	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/config"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/latencycheck"
	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/status"
)

func main() {
	virtClient, err := kubecli.GetKubevirtClient()
	if err != nil {
		log.Fatalf("Failed to obtain KubeVirt client %v", err)
	}

	resultsConfigMapNamespace, resultsConfigMapName, err := config.LoadResultsConfigMapNameFromEnv()
	if err != nil {
		log.Fatalln(err.Error())
	}

	reporter := status.NewConfigMapReporter(resultsConfigMapNamespace, resultsConfigMapName, virtClient)

	envVars, err := config.LoadEnvVars()
	if err != nil {
		reporter.Report(status.NewStatus(status.WithFailure(err)))
		log.Fatalln(err.Error())
	}

	options, err := latencycheck.CreateLatencyCheckOptions(envVars)
	if err != nil {
		reporter.Report(status.NewStatus(status.WithFailure(err)))
		log.Fatalln(err.Error())
	}

	result, err := latencycheck.StartLatencyCheck(virtClient, options)
	if err != nil {
		reporter.Report(status.NewStatus(
			status.WithFailure(err),
			status.WithResults(result.ConvertToStringsMap())))
		log.Fatalln(err.Error())
	}

	reporter.Report(status.NewStatus(
		status.WithSucceded(),
		status.WithResults(result.ConvertToStringsMap())))
}
