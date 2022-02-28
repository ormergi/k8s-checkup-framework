package checkup

import (
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

type Spec struct {
	image        string
	timeout      time.Duration
	params       map[string]string
	clusterRoles []string
	roles        []string
}

func NewSpecFromConfigMap(configMap *corev1.ConfigMap) (*Spec, error) {
	const (
		CheckupImageKey           = "spec.image"
		CheckupTimeoutKey         = "spec.timeout"
		CheckupParamNameKeyPrefix = "spec.param."
		CheckupClusterRolesKey    = "spec.clusterRoles"
		CheckupRolesKey           = "spec.roles"
	)

	spec := &Spec{}

	if configMap.Data == nil {
		return nil, fmt.Errorf("data field is missing from ConfigMap")
	}

	var err error
	var exists bool

	spec.image, exists = configMap.Data[CheckupImageKey]
	if !exists {
		return nil, fmt.Errorf("failed to read checkup image")
	}

	var rawTimeout string
	rawTimeout, exists = configMap.Data[CheckupTimeoutKey]
	if !exists {
		return nil, fmt.Errorf("failed to read checkup timeout")
	}

	spec.timeout, err = time.ParseDuration(rawTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse checkup timeout")
	}

	spec.params = make(map[string]string)
	for k, v := range configMap.Data {
		if strings.HasPrefix(k, CheckupParamNameKeyPrefix) && v != "" {
			paramName := strings.TrimPrefix(k, CheckupParamNameKeyPrefix)
			spec.params[paramName] = v
		}
	}

	if configMap.Data[CheckupClusterRolesKey] != "" {
		spec.clusterRoles = parseListSeparatedByNewlines(configMap.Data[CheckupClusterRolesKey])
	}

	if configMap.Data[CheckupRolesKey] != "" {
		spec.roles = parseListSeparatedByNewlines(configMap.Data[CheckupRolesKey])
	}

	return spec, nil
}

func (spec *Spec) Image() string {
	return spec.image
}

func (spec *Spec) Timeout() time.Duration {
	return spec.timeout
}

func (spec *Spec) Params() map[string]string {
	return spec.params
}

func (spec *Spec) ClusterRoles() []string {
	return spec.clusterRoles
}

func (spec *Spec) Roles() []string {
	return spec.roles
}

func parseListSeparatedByNewlines(rawString string) []string {
	trimmedString := strings.TrimSpace(rawString)
	return strings.Split(trimmedString, "\n")
}
