package status

import "fmt"

type status struct {
	succeeded     bool
	failureReason error
	results       map[string]string
}

func (s *status) ConvertToStringsMap() map[string]string {
	const (
		statusSucceededKey        = "status.succeeded"
		statusFailureReasonKey    = "status.failureReason"
		statusResultNameKeyPrefix = "status.result"
	)

	statusMap := map[string]string{}

	if s.failureReason != nil {
		statusMap[statusFailureReasonKey] = s.failureReason.Error()
	} else {
		statusMap[statusFailureReasonKey] = ""
	}

	statusMap[statusSucceededKey] = fmt.Sprintf("%v", s.succeeded)

	for k, v := range s.results {
		resultName := fmt.Sprintf("%s.%s", statusResultNameKeyPrefix, k)
		statusMap[resultName] = v
	}
	return statusMap
}

type option func(*status)

func NewStatus(opts ...option) *status {
	status := &status{}

	for _, fn := range opts {
		fn(status)
	}

	return status
}

func WithSucceded() option {
	return func(status *status) {
		status.succeeded = true
	}
}

func WithFailure(err error) option {
	return func(status *status) {
		status.succeeded = false
		status.failureReason = err
	}
}

func WithResults(results map[string]string) option {
	return func(status *status) {
		status.results = results
	}
}
