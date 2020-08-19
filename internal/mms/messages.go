package mms

// METDatasetCreatedMessage defines the message to send when a new dataset has been completed and persisted.
type METDatasetCreatedMessage struct {
	Name          string
	ReferenceTime string
}
