package framework

import (
	"github.com/orelmisan/k8s-checkup-framework/pkg/checkup"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	statusSucceededKey        = "status.succeeded"
	statusFailureReasonKey    = "status.failureReason"
	statusStartTimestamp      = "status.startTimestamp"
	statusCompletionTimestamp = "status.completionTimestamp"
	statusResultNameKeyPrefix = "status.result."
)

type Status struct {
	succeeded           string
	failureReason       string
	startTimestamp      string
	completionTimestamp string
	results             map[string]string
}

func NewStatus() *Status {
	return &Status{
		succeeded:     "",
		failureReason: "",
	}
}

func (s *Status) SetSucceeded(v bool) {
	s.succeeded = strconv.FormatBool(v)
}

func (s *Status) SetFailureReason(v string) {
	s.failureReason = v
}

func (s *Status) SetStartTimestampToNow() {
	s.startTimestamp = time.Now().Format(time.RFC3339)
}

func (s *Status) SetCompletionTimestampToNow() {
	s.completionTimestamp = time.Now().Format(time.RFC3339)
}

func (s *Status) UpdateFromCheckupStatus(checkupStatus *checkup.Status) {
	if s.succeeded == "" && checkupStatus.Succeeded() != "" {
		s.succeeded = checkupStatus.Succeeded()
	}

	if s.failureReason == "" && checkupStatus.FailureReason() != "" {
		s.failureReason = checkupStatus.FailureReason()
	}

	s.results = make(map[string]string)
	for k, v := range checkupStatus.Results() {
		s.results[k] = v
	}
}

func AppendStatusToFrameworkConfigMap(configMap *corev1.ConfigMap, status *Status) {
	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	configMap.Data[statusSucceededKey] = status.succeeded
	configMap.Data[statusFailureReasonKey] = status.failureReason
	configMap.Data[statusStartTimestamp] = status.startTimestamp
	configMap.Data[statusCompletionTimestamp] = status.completionTimestamp

	for k, v := range status.results {
		prefixedKey := strings.Join([]string{statusResultNameKeyPrefix, k}, "")
		configMap.Data[prefixedKey] = v
	}
}
