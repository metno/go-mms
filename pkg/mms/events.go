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
	"os"
	"os/user"
	"time"

	cenats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
)

// ProductEvent defines the message to send when a new Product has been completed and persisted.
// TODO: Find a proper name following our naming conventions: https://github.com/metno/MMS/wiki/Terminology
type ProductEvent struct {
	JobName         string
	Product         string // shortname, i.e., file(object) name without timestamp
	ProductLocation string // Storage system + protocol + filename or object name
	ProductionHub   string
	CreatedAt       time.Time // timestamp of the produced file (object)
	NextEventAt     time.Time // timestamp of the next event
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
	ceClient     cloudevents.Client
	cenatsSender cenats.Sender
}

// Generate a hub indetifier
func MakeHubIdentifier() (string, error) {
	var userName string

	hostName, err := os.Hostname()
	if err != nil {
		hostName = "unkown"
	}

	user, err := user.Current()
	if err == nil {
		userName = user.Username
	} else {
		userName = "unkown"
	}

	return fmt.Sprintf("%s@%s", userName, hostName), nil
}

// NewNatsConsumerClient creates a cloudevent client for consuming MMS events from NATS.
func NewNatsConsumerClient(natsURL string) (*EventClient, error) {
	eClient, err := newNATSConsumer(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to events: %v", err)
	}

	return &EventClient{
		ceClient: eClient,
	}, nil
}

// NewNatsSenderClient creates a cloudevent client for sending MMS events to NATS.
func NewNatsSenderClient(natsURL string) (*EventClient, error) {
	eClient, pEvent, err := newNATSSender(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to events: %v", err)
	}

	return &EventClient{
		ceClient:     eClient,
		cenatsSender: pEvent,
	}, nil
}

// WatchProductEvents will call your callback function on each incoming event from the MMS Nats server.
func (eClient *EventClient) WatchProductEvents(callback ProductEventCallback, opts Options) {
	for {
		if err := eClient.ceClient.StartReceiver(context.Background(), productReceiver(callback)); err != nil {
			log.Printf("failed to start nats receiver, %s", err.Error())
		}
	}
}

// ListProductEvents will give all available events from the specified events cache.
func ListProductEvents(apiURL string, opts Options) ([]*ProductEvent, error) {
	events := []*ProductEvent{}

	resp, err := http.Get(apiURL)
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
func MakeProductEvent(natsURL string, pEvent *ProductEvent) error {

	mmsClient, err := NewNatsSenderClient(natsURL)
	if err != nil {
		return fmt.Errorf("failed to post event to messaging service: %v", err)
	}

	err = mmsClient.PostProductEvent(pEvent, Options{})
	if err != nil {
		return fmt.Errorf("failed to post event to messaging service: %v", err)
	}

	mmsClient.cenatsSender.Close(context.Background())

	return nil
}

// PostProductEvent generates an event and sends it to the specified messaging service.
func (eClient *EventClient) PostProductEvent(pEvent *ProductEvent, opts Options) error {
	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetType("no.met.mms.product.v1")
	event.SetTime(time.Now())
	event.SetSource(pEvent.ProductionHub)
	event.SetSubject(pEvent.Product)

	err := event.SetData("application/json", pEvent)
	if err != nil {
		return fmt.Errorf("failed to properly encode event data for product event: %v", err)
	}

	if result := eClient.ceClient.Send(context.Background(), event); cloudevents.IsUndelivered(result) {
		return fmt.Errorf("failed to send: %v", result.Error())
	}

	return nil
}

func newNATSSender(natsURL string) (cloudevents.Client, cenats.Sender, error) {
	pEvent, err := cenats.NewSender(natsURL, "mms", cenats.NatsOptions())
	if err != nil {
		return nil, cenats.Sender{}, fmt.Errorf("failed to create nats protocol: %v", err)
	}

	eClient, err := cloudevents.NewClient(pEvent)
	if err != nil {
		return nil, cenats.Sender{}, fmt.Errorf("failed to create client, %v", err)
	}

	return eClient, *pEvent, nil
}

func newNATSConsumer(natsURL string) (cloudevents.Client, error) {
	pEvent, err := cenats.NewConsumer(natsURL, "mms", cenats.NatsOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to create nats protocol, %v", err)
	}

	eClient, err := cloudevents.NewClient(pEvent)
	if err != nil {
		return nil, fmt.Errorf("failed to create client, %v", err)
	}

	return eClient, nil
}

func productReceiver(callback ProductEventCallback) func(context.Context, cloudevents.Event) error {
	return func(ctx context.Context, event cloudevents.Event) error {
		mmsEvent := ProductEvent{}

		if err := event.DataAs(&mmsEvent); err != nil {
			return fmt.Errorf("failed to decode event as product event: %v", err)
		}

		return callback(&mmsEvent)
	}
}
