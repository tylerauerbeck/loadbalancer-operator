package cmd

import "errors"

var (
	// ErrNATSURLRequired is returned when a NATS url is missing
	ErrNATSURLRequired = errors.New("nats url is required and cannot be empty")
	// ErrNATSSubjectPrefix is returned when a NATS subject prefix is missing
	ErrNATSSubjectPrefix = errors.New("nats subject prefix is required and cannot be empty")
	// ErrNATSStreamName is returned when a NATS Stream Name is missing
	ErrNATSStreamName = errors.New("nats stream name is required and cannot be empty")
	// ErrChartPath is returned when a Helm chart path is missing
	errChartPath         = errors.New("chart path is required and cannot be empty")
	errRequiredTopics    = errors.New("at least one topic is required")
	errInvalidKubeClient = errors.New("failed to create kubernetes client")
	errInvalidHelmChart  = errors.New("failed to load helm chart")
)
