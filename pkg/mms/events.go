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
	"github.com/google/uuid"
)

// ProductEvent defines the message to send when a new Product has been completed and persisted.
// TODO: Find a proper name following our naming conventions: https://github.com/metno/MMS/wiki/Terminology
type ProductEvent struct {
	Product         string
	ProductionHub   string
	ProductSlug     string
	CreatedAt       time.Time
	ProductLocation url.URL
}

// Options defines the filtering options you can set to limit what kinds of events you will receive.
type Options struct {
	Product       string
	ProductionHub string
}

// ProductEventCallback specifies the function signature for receiving ProductEvent events.
type ProductEventCallback func(e *ProductEvent) error

// WatchProductEvents will call your callback function on each incoming event from the MMS Nats server.
func WatchProductEvents(natsURL string, opts Options, callback ProductEventCallback) error {
	c, err := newNATSConsumer(natsURL)
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %v", err)
	}

	for {
		if err := c.StartReceiver(context.Background(), productReceiver(callback)); err != nil {
			log.Printf("failed to start nats receiver, %s", err.Error())
		}
	}
}

// ListProductEvents will give all available events from the specified events cache.
func ListProductEvents(eventCache string, opts Options) ([]*ProductEvent, error) {
	events := []*ProductEvent{}

	resp, err := http.Get(eventCache)
	if err != nil {
		return nil, fmt.Errorf("Could not get events from local http server:%v", err)
	}
	defer resp.Body.Close()

	event := cloudevents.NewEvent()
	if err = json.NewDecoder(resp.Body).Decode(&event); err != nil {
		return nil, fmt.Errorf("failed to decode event: %v", err)
	}

	eventData := ProductEvent{}
	if err = event.DataAs(&eventData); err != nil {
		return nil, fmt.Errorf("failed to decode message in event: %v", err)
	}
	eventData.ProductionHub = event.Source()

	events = append(events, &eventData)

	return events, nil
}

// PostProductEvent generates an event and sends it to the specified messaging service.
func PostProductEvent(natsURL string, opts Options, p *ProductEvent) error {
	c, err := newNATSSender(natsURL)
	if err != nil {
		return fmt.Errorf("can not send event to messaging service: %v", err)
	}

	e := cloudevents.NewEvent()
	e.SetID(uuid.New().String())
	e.SetType("no.met.mms.product.v1")
	e.SetTime(time.Now())
	e.SetSource(p.ProductionHub)
	e.SetSubject(p.ProductSlug)

	err = e.SetData("application/json", p)
	if err != nil {
		return fmt.Errorf("failed to properly encode event data for product event: %v", err)
	}

	if result := c.Send(context.Background(), e); cloudevents.IsUndelivered(result) {
		return fmt.Errorf("failed to send: %v", result.Error())
	}

	// FIXME(havardf): Weird race condition with closing connection and actually getting the event sent. Figure out how this actually should be done robustly.
	time.Sleep(50 * time.Millisecond)

	return nil
}

func newNATSSender(natsURL string) (cloudevents.Client, error) {
	p, err := cenats.NewSender(natsURL, "mms", cenats.NatsOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to create nats protocol: %v", err)
	}

	c, err := cloudevents.NewClient(p)
	if err != nil {
		return nil, fmt.Errorf("failed to create client, %v", err)
	}

	return c, nil
}

func newNATSConsumer(natsURL string) (cloudevents.Client, error) {
	p, err := cenats.NewConsumer(natsURL, "mms", cenats.NatsOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to create nats protocol, %v", err)
	}

	c, err := cloudevents.NewClient(p)
	if err != nil {
		return nil, fmt.Errorf("failed to create client, %v", err)
	}

	return c, nil
}

func productReceiver(callback ProductEventCallback) func(context.Context, cloudevents.Event) error {
	return func(ctx context.Context, e cloudevents.Event) error {
		mmsEvent := ProductEvent{}

		if err := e.DataAs(&mmsEvent); err != nil {
			return fmt.Errorf("failed to decode event as product event: %v", err)
		}

		return callback(&mmsEvent)
	}
}
