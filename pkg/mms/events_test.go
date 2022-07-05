/*
Copyright 2020–2021 MET Norway

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package mms

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol/gochan"
)

var erroneousEventData = `
{
	"data": {
	"ProductionHub": "ecflow.modellprod",
	"Product": "arome_arctic_sfx_2_5km",
	"CreatedAt": "2020-08-26T12:18:48.281847242+02:00",
	"ProductLocation": ""
	},
	"datacontenttype": "application/json",
	"id": "0173c5ce-e1fb-11ea-9c78-6b708419aa07",
	"source": "ecflow/modellprod",
	"specversion": "1.0",
	"subject": "arome.arctic",
	"type": "no.met.Product.created.v1"
}`

var correctEventData = `
[{
	"Product": "Arome Arctic",
	"ProductionHub": "ecflow.modellprod",
	"Product": "arome_arctic_sfx_2_5km",
	"CreatedAt": "2020-08-26T12:18:48.281847242+02:00",
	"ProductLocation": ""
}]`

func TestProductEvent(t *testing.T) {
	eventData := ProductEvent{}
	err := json.Unmarshal([]byte(erroneousEventData), &eventData)

	if err != nil || eventData.Product != "" {

		t.Errorf("Expected missing Product field; Got %v", eventData.Product)
	}
	fmt.Println(err)
}

func TestListProductEvents(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, correctEventData)
	}))

	list, err := ListProductEvents(ts.URL)
	if err != nil {
		t.Errorf("Expected no errors; Got %v", err)
	}

	if len(list) != 1 {
		t.Errorf("Expected 1 event; Got %d events", len(list))
	}

	if list[0].Product != "arome_arctic_sfx_2_5km" {
		t.Errorf("Expected Product field value 'arome_arctic_sfx_2_5km'; Got %s", list[0].Product)
	}
}

func TestPostProductEvent(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	productEvent := ProductEvent{
		JobName:         "test-job",
		Product:         "test-product",
		ProductLocation: ".",
		ProductionHub:   ts.URL,
		Counter:         1,
		TotalCount:      1,
		RefTime:         PEventTime(time.Date(1918, 10, 28, 12, 00, 00, 00, time.UTC)),
		CreatedAt:       PEventTime(time.Now()),
		NextEventAt:     PEventTime(time.Now().Add(time.Second * time.Duration(3600))),
	}

	err := PostProductEvent(ts.URL, "no-api-key", &productEvent, false)
	if err != nil {
		t.Errorf("Expected no errors; Got %v", err)
	}
}

func TestPostProductEventNotSuccessful(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))

	productEvent := ProductEvent{
		JobName:         "test-job",
		Product:         "test-product",
		ProductLocation: ".",
		ProductionHub:   ts.URL,
		Counter:         1,
		TotalCount:      1,
		RefTime:         PEventTime(time.Date(1918, 10, 28, 12, 00, 00, 00, time.UTC)),
		CreatedAt:       PEventTime(time.Now()),
		NextEventAt:     PEventTime(time.Now().Add(time.Second * time.Duration(3600))),
	}

	err := PostProductEvent(ts.URL, "no-api-key", &productEvent, false)
	if err == nil {
		t.Errorf("Expected the function to return an error.")
	}
}

func EmitProductEventMessage(t *testing.T) {
	eClient := newMockCloudeventsClient()

	event := ProductEvent{ProductionHub: "test-hub", Product: "test"}
	err := eClient.EmitProductEventMessage(&event)

	if err != nil {
		t.Errorf("Expected no errors; Got this error: %s", err)
	}
}

// EventClient that sends and receives events on an internal go channel.
func newMockCloudeventsClient() *EventClient {
	cEvent, err := cloudevents.NewClient(gochan.New())
	if err != nil {
		log.Fatalln("Failed to create event gochan mock cloudevents client.")
	}

	// Start the receiver
	go func() {
		if err := cEvent.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) {
			log.Printf("[receiver] %s", event)
		}); err != nil && err.Error() != "context deadline exceeded" {
			log.Fatalf("[receiver] start receiver returned an error: %s", err)
		}
		log.Println("[receiver] stopped")
	}()

	return &EventClient{
		ceClient: cEvent,
	}
}
