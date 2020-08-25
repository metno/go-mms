package mms

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	cenats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
)

// DatasetCreatedEvent defines the message to send when a new dataset has been completed and persisted.
// TODO: Find a proper name following our naming conventions: https://github.com/metno/MMS/wiki/Terminology
type DatasetCreatedEvent struct {
	Product         string
	ProductionHub   string
	ProductSlug     string
	CreatedAt       time.Time
	DatasetLocation url.URL
}

// Options defines the filtering options you can set to limit what kinds of events you will receive.
type Options struct {
	Product       string
	ProductionHub string
}

// DatasetCreatedEventCallback specifies the function signature for receiving DatasetCreatedEvent events.
type DatasetCreatedEventCallback func(e *DatasetCreatedEvent) error

// WatchDatasetCreatedEvents will call your callback function on each incoming event from the MMS Nats server.
func WatchDatasetCreatedEvents(natsURL string, opts Options, callback DatasetCreatedEventCallback) error {
	c, err := newNATSConsumer(natsURL)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %v", err)
	}

	for {
		if err := c.StartReceiver(context.Background(), datasetCreatedReceiver(callback)); err != nil {
			log.Printf("failed to start nats receiver, %s", err.Error())
		}
	}
}

// ListDatasetCreatedEvents will give all available events from the specified events cache.
func ListDatasetCreatedEvents(url string, opts Options) ([]*DatasetCreatedEvent, error) {
	resp, err := http.Get(url)
	if err != nil {
		log.Fatalf("Could not get events from local http server:%v", err)
	}
	defer resp.Body.Close()

	event := cloudevents.NewEvent()
	if err = json.NewDecoder(resp.Body).Decode(&event); err != nil {
		log.Fatalf("Failed to decode event: %v", err)
	}

	eventData := DatasetCreatedEvent{}
	if err = event.DataAs(&eventData); err != nil {
		log.Fatalf("Failed to decode message in event: %v", err)
	}
	eventData.ProductionHub = event.Source()

	return []*DatasetCreatedEvent{&eventData}, nil
}

func newNATSConsumer(natsURL string) (cloudevents.Client, error) {
	ctx := context.Background()

	p, err := cenats.NewConsumer(natsURL, "test", cenats.NatsOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to create nats protocol, %v", err)
	}

	defer p.Close(ctx)

	c, err := cloudevents.NewClient(p)
	if err != nil {
		return nil, fmt.Errorf("failed to create client, %v", err)
	}

	return c, nil

}

func datasetCreatedReceiver(callback DatasetCreatedEventCallback) func(context.Context, cloudevents.Event) error {
	return func(ctx context.Context, e cloudevents.Event) error {
		mmsEvent := DatasetCreatedEvent{}

		return callback(&mmsEvent)
	}
}
