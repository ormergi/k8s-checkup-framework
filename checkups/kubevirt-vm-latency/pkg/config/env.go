package config

import (
	"fmt"
	"os"
)

const (
	ResultsConfigMapNamespaceEnvVarName     = "RESULT_CONFIGMAP_NAMESPACE"
	ResultsConfigMapNameEnvVarName          = "RESULT_CONFIGMAP_NAME"
	NetworkNamespaceEnvVarName              = "NAD_NAMESPACE"
	NetworkNameEnvVarName                   = "NAD_NAME"
	SampleDurationSecondsEnvVarName         = "SAMPLE_DURATION_SECONDS"
	SourceNodeNameEnvVarName                = "SOURCE_NODE"
	TargetNodeNameEnvVarName                = "TARGET_NODE"
	DesiredMaxLatencyMillisecondsEnvVarName = "MAX_DESIRED_LATENCY_MILLISECONDS"
)

const errMsgFormat = "failed to load %s environment variable"

func LoadResultsConfigMapNameFromEnv() (string, string, error) {
	var exists bool

	namespace, exists := os.LookupEnv(ResultsConfigMapNamespaceEnvVarName)
	if !exists {
		return "", "", fmt.Errorf(errMsgFormat, ResultsConfigMapNamespaceEnvVarName)
	}

	name, exists := os.LookupEnv(ResultsConfigMapNameEnvVarName)
	if !exists {
		return "", "", fmt.Errorf(errMsgFormat, ResultsConfigMapNameEnvVarName)
	}

	return namespace, name, nil
}

func LoadEnvVars() (map[string]string, error) {
	var exists bool

	mandatoryEnvVarNames := []string{
		NetworkNamespaceEnvVarName,
		NetworkNameEnvVarName,
	}

	envVars := map[string]string{}
	for _, envVarName := range mandatoryEnvVarNames {
		envVars[envVarName], exists = os.LookupEnv(envVarName)
		if !exists {
			return envVars, fmt.Errorf(errMsgFormat, envVarName)
		}
	}

	optionalEnvVarNames := []string{
		SampleDurationSecondsEnvVarName,
		SourceNodeNameEnvVarName,
		TargetNodeNameEnvVarName,
		DesiredMaxLatencyMillisecondsEnvVarName,
	}
	for _, envVarName := range optionalEnvVarNames {
		envVars[envVarName] = os.Getenv(envVarName)
	}

	return envVars, nil
}
