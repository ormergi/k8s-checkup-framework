package nads

import (
	"context"
	"encoding/json"
	"fmt"

	k8scni "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	k8smetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"kubevirt.io/client-go/kubecli"
)

func Get(client kubecli.KubevirtClient, namespace, name string) (*k8scni.NetworkAttachmentDefinition, error) {
	return client.NetworkClient().
		K8sCniCncfIoV1().
		NetworkAttachmentDefinitions(namespace).
		Get(context.Background(), name, k8smetav1.GetOptions{})
}

const (
	NetworkConfigTypeFiledName    = "type"
	NetworkConfigPluginsFiledName = "plugins"
)

func ConfigCniPlugins(nad *k8scni.NetworkAttachmentDefinition) ([]string, error) {
	var netConf map[string]interface{}
	err := json.Unmarshal([]byte(nad.Spec.Config), &netConf)
	if err != nil {
		return nil, fmt.Errorf("failed to get netwrok type: failed to unmarshal NetworkAttachmentDefinitions config: %v", err)
	}

	var cniPlugins []string
	// Identify target is single CNI config or plugins
	if pluginsRaw, exists := netConf[NetworkConfigPluginsFiledName]; exists {
		// CNI conflist
		plugins := pluginsRaw.([]interface{})
		for _, pluginRaw := range plugins {
			plugin := pluginRaw.(map[string]interface{})
			pluginType := fmt.Sprintf("%v", plugin[NetworkConfigTypeFiledName])
			cniPlugins = append(cniPlugins, pluginType)
		}
	} else {
		// single CNI config
		cniPlugins = append(cniPlugins, fmt.Sprintf("%v", netConf[NetworkConfigTypeFiledName]))
	}

	return cniPlugins, nil
}
