/*
  Copyright 2020 MET Norway

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
	"time"

	cenats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

// ProductEvent defines the message to send when a new Product has been completed and persisted.
// TODO: Find a proper name following our naming conventions: https://github.com/metno/MMS/wiki/Terminology
type ProductEvent struct {
	Product         string // shortname, i.e., file(object) name without timestamp
	ProductionHub   string
	CreatedAt       time.Time // timestamp of the produced file (object)
	ProductLocation string    //storage system + protocol + filename or object name
}

// Options defines the filtering options you can set to limit what kinds of events you will receive.
type Options struct {
	Product       string
	ProductionHub string
}

// ProductEventCallback specifies the function signature for receiving ProductEvent events.
type ProductEventCallback func(e *ProductEvent) error

// EventClient defines the MMS client used to send and receive events from the MMS messaging service.
type EventClient struct {
	ce cloudevents.Client
}

// ProductionHub specifies the available hubs
type ProductionHub struct {
	Name       string
	NatsURL    string
	EventCache string
}

// ListProductionHubs to contact
func ListProductionHubs() []ProductionHub {
	return []ProductionHub{
		{
			Name:       "test-hub",
			NatsURL:    "nats://localhost:4222",
			EventCache: "http://localhost:8080",
		},
	}
}

// Generate an event identifier
func MakeClientIdentifier() (string, error) {
	return "test", nil
}

// NewNatsConsumerClient creates a cloudevent client for consuming MMS events from NATS.
func NewNatsConsumerClient(natsURL string) (*EventClient, error) {
	c, err := newNATSConsumer(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to events: %v", err)
	}

	return &EventClient{
		ce: c,
	}, nil
}

// NewNatsSenderClient creates a cloudevent client for sending MMS events to NATS.
func NewNatsSenderClient(natsURL string) (*EventClient, error) {
	c, err := newNATSSender(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to events: %v", err)
	}

	return &EventClient{
		ce: c,
	}, nil
}

// WatchProductEvents will call your callback function on each incoming event from the MMS Nats server.
func (c *EventClient) WatchProductEvents(callback ProductEventCallback, opts Options) {
	for {
		if err := c.ce.StartReceiver(context.Background(), productReceiver(callback)); err != nil {
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

// MakeProductEvent prepares and sends the product event
func MakeProductEvent(hubs []ProductionHub, p *ProductEvent) error {
	var hub ProductionHub
	for _, h := range hubs {
		if h.Name == p.ProductionHub {
			hub = h
			break
		}
	}

	if (hub == ProductionHub{}) {
		return fmt.Errorf("could not find correct hub to send event")
	}

	mmsClient, err := NewNatsSenderClient(hub.NatsURL)
	if err != nil {
		return fmt.Errorf("failed to post event to messaging service: %v", err)
	}

	err = mmsClient.PostProductEvent(p, Options{})
	if err != nil {
		return fmt.Errorf("failed to post event to messaging service: %v", err)
	}

	return nil
}

// PostProductEvent generates an event and sends it to the specified messaging service.
func (c *EventClient) PostProductEvent(p *ProductEvent, opts Options) error {
	e := cloudevents.NewEvent()
	e.SetID(uuid.New().String())
	e.SetType("no.met.mms.product.v1")
	e.SetTime(time.Now())
	e.SetSource(p.ProductionHub)
	e.SetSubject(p.Product)

	err := e.SetData("application/json", p)
	if err != nil {
		return fmt.Errorf("failed to properly encode event data for product event: %v", err)
	}

	if result := c.ce.Send(context.Background(), e); cloudevents.IsUndelivered(result) {
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
