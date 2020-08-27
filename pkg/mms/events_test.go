package mms

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/protocol/gochan"
)

var erroneousEventData = `
{
	"data": {
	"ProductionHub": "ecflow.modellprod",
	"ProductSlug": "arome.arctic",
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
{
	"data": {
	"Product": "Arome Arctic",
	"ProductionHub": "ecflow.modellprod",
	"ProductSlug": "arome.arctic",
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

	list, err := ListProductEvents(ts.URL, Options{})
	if err != nil {
		t.Errorf("Expected no errors; Got %v", err)
	}

	if len(list) != 1 {
		t.Errorf("Expected 1 event; Got %d events", len(list))
	}

	if list[0].Product != "Arome Arctic" {
		t.Errorf("Expected Product field value 'Arome Arctic'; Got %s", list[0].Product)
	}
}

// TODO
// func TestPostProductEvent(t *testing.T) {
// 	c := newMockCloudeventsClient()

// 	event := ProductEvent{}

// 	err := c.PostProductEvent(&event, Options{})

// 	if err != nil {
// 		t.Errorf("Expected no errors; Got this error: %s", err)
// 	}
// }

// EventClient that sends and receives events on an internal go channel.
func newMockCloudeventsClient() *EventClient {
	c, err := cloudevents.NewClient(gochan.New())
	if err != nil {
		log.Fatalln("Failed to create event gochan mock cloudevents client.")
	}

	// Start the receiver
	go func() {
		if err := c.StartReceiver(context.Background(), func(ctx context.Context, event cloudevents.Event) {
			log.Printf("[receiver] %s", event)
		}); err != nil && err.Error() != "context deadline exceeded" {
			log.Fatalf("[receiver] start receiver returned an error: %s", err)
		}
		log.Println("[receiver] stopped")
	}()

	return &EventClient{
		ce: c,
	}
}
