package mms

// METDatasetCreatedEvent defines the message to send when a new dataset has been completed and persisted.
type METDatasetCreatedEvent struct {
	Name          string
	ReferenceTime string
}
