package mms

import (
	"encoding/json"
	"fmt"
	"testing"
)

var erroneousEventData = `
{
	"data": {
		"Name": "Arome Arctic"
	},
	"datacontenttype": "application/json",
	"id": "0173c5ce-e1fb-11ea-9c78-6b708419aa07",
	"source": "ecflow/modellprod",
	"specversion": "1.0",
	"type": "no.met.dataset.created.v1"
}`

func TestDatasetCreatedEvent(t *testing.T) {
	eventData := METDatasetCreatedEvent{}
	err := json.Unmarshal([]byte(erroneousEventData), &eventData)

	if err != nil || eventData.ReferenceTime != "" {

		t.Errorf("Expected missing ReferenceTime field; Got %v", eventData.ReferenceTime)
	}
	fmt.Println(err)
}
