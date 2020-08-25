package mms

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
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
	eventData := DatasetCreatedEvent{}
	err := json.Unmarshal([]byte(erroneousEventData), &eventData)

	if (err != nil || eventData.CreatedAt != time.Time{}) {

		t.Errorf("Expected missing CreatedAt field; Got %v", eventData.CreatedAt)
	}
	fmt.Println(err)
}
