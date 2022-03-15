package status

import (
	"encoding/json"
	"log"

	"kubevirt.io/client-go/kubecli"

	"github.com/orelmisan/k8s-checkup-framework/checkups/kubevirt-vm-latency/pkg/utils/configmaps"
)

type checkupStatusReporter struct {
	name      string
	namespace string
	client    kubecli.KubevirtClient
}

func NewConfigMapReporter(namespace, name string, client kubecli.KubevirtClient) *checkupStatusReporter {
	return &checkupStatusReporter{
		namespace: namespace,
		name:      name,
		client:    client,
	}
}

func (r *checkupStatusReporter) Report(status *status) {
	statusMap := status.ConvertToStringsMap()

	rawStatus, err := json.MarshalIndent(statusMap, "", " ")
	if err != nil {
		log.Printf("failed to marashl status: %v", err)
	}
	log.Println(string(rawStatus))

	if err := configmaps.AppendData(r.client, r.namespace, r.name, statusMap); err != nil {
		log.Printf("failed to report status: %v", err)
	}
}
