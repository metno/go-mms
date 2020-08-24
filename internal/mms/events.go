package mms

// METDatasetCreatedEvent defines the message to send when a new dataset has been completed and persisted.
// TODO: Find a proper name following our naming conventions: https://github.com/metno/MMS/wiki/Terminology
type METDatasetCreatedEvent struct {
	Name          string
	ReferenceTime string
}
