package checkup

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

const (
	statusSucceededKey        = "status.succeeded"
	statusFailureReasonKey    = "status.failureReason"
	statusResultNameKeyPrefix = "status.result."
)

type Status struct {
	succeeded     string
	failureReason string
	results       map[string]string
}

func newStatusFromConfigMap(configMap *corev1.ConfigMap) (*Status, error) {
	if configMap.Data == nil {
		return nil, fmt.Errorf("data field is missing from ConfigMap")
	}

	status := &Status{}

	var exists bool
	var succeeded string

	succeeded, exists = configMap.Data[statusSucceededKey]
	if !exists {
		errString := "failed to read succeeded field"
		status.succeeded = "false"
		status.failureReason = errString

		return status, nil
	}

	if succeeded != "true" && succeeded != "false" {
		errString := "illegal value in succeeded field"
		status.succeeded = "false"
		status.failureReason = errString

		return status, nil
	}
	status.succeeded = succeeded

	status.failureReason, exists = configMap.Data[statusFailureReasonKey]
	if !exists {
		status.failureReason = "Unknown"
	}

	status.results = make(map[string]string)
	for k, v := range configMap.Data {
		if strings.HasPrefix(k, statusResultNameKeyPrefix) && v != "" {
			resultName := strings.TrimPrefix(k, statusResultNameKeyPrefix)
			status.results[resultName] = v
		}
	}

	return status, nil
}

func (s *Status) Succeeded() string {
	return s.succeeded
}

func (s *Status) FailureReason() string {
	return s.failureReason
}

func (s *Status) Results() map[string]string {
	return s.results
}
