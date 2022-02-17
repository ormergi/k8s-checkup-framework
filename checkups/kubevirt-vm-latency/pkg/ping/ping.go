package ping

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	expect "github.com/google/goexpect"
)

type Result struct {
	min                 time.Duration
	max                 time.Duration
	average             time.Duration
	jitter              time.Duration
	measurementDuration time.Duration
}

func (r *Result) Max() time.Duration {
	return r.max
}

func (r *Result) SetMeasurementDuration(measurementDuration time.Duration) {
	r.measurementDuration = measurementDuration
}

func (r *Result) ConvertToStringsMap() map[string]string {
	const (
		minLatencyKey          = "status.result.minLatencyNanoseconds"
		maxLatencyKey          = "status.result.maxLatencyNanoseconds"
		averageLatencyKey      = "status.result.averageLatencyNanoseconds"
		jitterKey              = "status.result.jitterNanoseconds"
		measurementDurationKey = "status.result.measurementDurationSeconds"
	)

	resultMap := map[string]string{}

	resultMap[minLatencyKey] = fmt.Sprintf("%d", r.min.Nanoseconds())
	resultMap[maxLatencyKey] = fmt.Sprintf("%d", r.max.Nanoseconds())
	resultMap[averageLatencyKey] = fmt.Sprintf("%d", r.average.Nanoseconds())
	resultMap[jitterKey] = fmt.Sprintf("%d", r.jitter.Nanoseconds())
	resultMap[measurementDurationKey] = fmt.Sprintf("%.2f", r.measurementDuration.Seconds())
	return resultMap
}

func ParsePingResult(pingResult []expect.BatchRes) Result {
	var result Result
	latencyPattern := regexp.MustCompile(`(round-trip|rtt)\s+\S+\s*=\s*([0-9.]+)/([0-9.]+)/([0-9.]+)/([0-9.]+)\s*ms`)

	for _, response := range pingResult {
		matches := latencyPattern.FindAllStringSubmatch(response.Output, -1)
		for _, item := range matches {
			min, err := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[2])))
			if err != nil {
				log.Printf("failed to parse min latency from result: %v", err)
			}
			result.min = min

			average, err := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[3])))
			if err != nil {
				log.Printf("failed to parse average jitter from result: %v", err)
			}
			result.average = average

			max, err := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[4])))
			if err != nil {
				log.Printf("failed to parse max latency from result: %v", err)
			}
			result.max = max

			jitter, _ := time.ParseDuration(fmt.Sprintf("%sms", strings.TrimSpace(item[5])))
			if err != nil {
				log.Printf("failed to parse jitter from result: %v", err)
			}
			result.jitter = jitter
		}
	}

	return result
}
