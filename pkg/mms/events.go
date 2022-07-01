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
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"

	cenats "github.com/cloudevents/sdk-go/protocol/nats/v2"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// ProductEvent defines the message to send when a new Product has been completed and persisted.
// TODO: Find a proper name following our naming conventions: https://github.com/metno/MMS/wiki/Terminology
type ProductEvent struct {
	JobName string `env:"MMS_PRODUCT_EVENT_JOB_NAME"`
	// shortname, i.e., file(object) name without timestamp
	Product string `env:"MMS_PRODUCT_EVENT_PRODUCT,required=true"`
	// Storage system + protocol + filename or object name
	ProductLocation string    `env:"MMS_PRODUCT_EVENT_PRODUCT_LOCATION"`
	ProductionHub   string    `env:"MMS_PRODUCT_EVENT_PRODUCTION_HUB"`
	MMD             string    `env:"MMS_PRODUCT_EVENT_MMD"` // MMDfile for the product
	Counter         int       `env:"MMS_PRODUCT_EVENT_COUNTER"`
	TotalCount      int       `env:"MMS_PRODUCT_EVENT_TOTAL_COUNT"`
	RefTime         time.Time `env:"MMS_PRODUCT_EVENT_REF_TIME"`      // Reference time
	CreatedAt       time.Time `env:"MMS_PRODUCT_EVENT_CREATED_AT"`    // timestamp of the produced file (object)
	NextEventAt     time.Time `env:"MMS_PRODUCT_EVENT_NEXT_EVENT_AT"` // timestamp of the next event
}

type HeartBeatEvent struct {
	ProductionHub string
	CreatedAt     time.Time // timestamp of the produced file (object)
	NextEventAt   time.Time // timestamp of the next event
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
func NewNatsConsumerClient(natsURL string, natsCredentials nats.Option) (*EventClient, error) {
	eClient, err := newNATSConsumer(natsURL, natsCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to events: %v", err)
	}

	return &EventClient{
		ceClient: eClient,
	}, nil
}

// NewNatsSenderClient creates a cloudevent client for sending MMS events to NATS.
func NewNatsSenderClient(natsURL string, natsCredentials nats.Option) (*EventClient, error) {
	eClient, pEvent, err := newNATSSender(natsURL, natsCredentials)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to events: %v", err)
	}

	return &EventClient{
		ceClient:     eClient,
		cenatsSender: pEvent,
	}, nil
}

// WatchProductEvents will call your callback function on each incoming event from the MMS Nats server.
func (eClient *EventClient) WatchProductEvents(callback ProductEventCallback) {
	for {
		if err := eClient.ceClient.StartReceiver(context.Background(), productReceiver(callback)); err != nil {
			log.Printf("failed to start nats receiver, %s", err.Error())
		}
	}
}

// ListProductEvents will give all available events from the specified events cache.
func ListProductEvents(apiURL string) ([]*ProductEvent, error) {

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("Could not get events from local http server:%v", err)
	}
	defer resp.Body.Close()

	events := []*ProductEvent{}
	err = json.NewDecoder(resp.Body).Decode(&events)
	if err != nil {
		return nil, fmt.Errorf("failed to decode event: %v", err)
	}

	return events, nil
}

// MakeProductEvent prepares and sends the product event
func MakeProductEvent(natsURL string, natsCredentials nats.Option, pEvent *ProductEvent) error {

	mmsClient, err := NewNatsSenderClient(natsURL, natsCredentials)
	if err != nil {
		return fmt.Errorf("failed to create messaging service: %v", err)
	}

	err = mmsClient.EmitProductEventMessage(pEvent)
	if err != nil {
		return fmt.Errorf("failed to post product to messaging service: %v", err)
	}

	mmsClient.cenatsSender.Close(context.Background())

	return nil
}

func PostProductEvent(mmsdURL string, apiKey string, pe *ProductEvent, insecure bool) error {
	var err error

	url := mmsdURL + "/api/v1/events"

	jsonStr, err := json.Marshal(&pe)
	if err != nil {
		return fmt.Errorf("failed to marshal ProductEvent: %v", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}

	httpReq.Header.Set("Api-Key", apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	var tr *http.Transport
	if insecure {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		tr = &http.Transport{}
	}

	httpClient := &http.Client{Transport: tr}
	httpResp, err := httpClient.Do(httpReq)

	if err != nil {
		return fmt.Errorf("failed to create http client: %v", err)
	}

	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusCreated {
		return fmt.Errorf("POST to %s failed with status: %s", url, httpResp.Status)
	}
	return nil
}

// MakeProductEvent prepares and sends the product event
func MakeHeartBeatEvent(natsURL string, natsCredentials nats.Option, hEvent *HeartBeatEvent) error {

	mmsClient, err := NewNatsSenderClient(natsURL, natsCredentials)
	if err != nil {
		return fmt.Errorf("failed to create messaging service: %v", err)
	}

	err = mmsClient.EmitHeartBeatMessage(hEvent)
	if err != nil {
		return fmt.Errorf("failed to post heartbeat to messaging service: %v", err)
	}

	mmsClient.cenatsSender.Close(context.Background())

	return nil
}

// EmitProductEventMessage generates an event and sends it to the specified messaging service.
func (eClient *EventClient) EmitProductEventMessage(pEvent *ProductEvent) error {
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
		log.Fatalf(result.Error())
		return fmt.Errorf("failed to send: %v", result.Error())
	}

	return nil
}

// EmitHeartBeatMessage generates an event and sends it to the specified messaging service.
func (eClient *EventClient) EmitHeartBeatMessage(hEvent *HeartBeatEvent) error {
	event := cloudevents.NewEvent()
	event.SetID(uuid.New().String())
	event.SetType("no.met.mms.heartbeat.v1")
	event.SetTime(time.Now())
	event.SetSource("heartBeat")
	event.SetSubject("heartBeat")

	err := event.SetData("application/json", hEvent)
	if err != nil {
		return fmt.Errorf("failed to properly encode event data for heartbeat event: %v", err)
	}

	if result := eClient.ceClient.Send(context.Background(), event); cloudevents.IsUndelivered(result) {
		return fmt.Errorf("failed to send: %v", result.Error())
	}

	return nil
}

func newNATSSender(natsURL string, natsCredentials nats.Option) (cloudevents.Client, cenats.Sender, error) {
	pEvent, err := cenats.NewSender(natsURL, "mms", cenats.NatsOptions(natsCredentials))
	if err != nil {
		return nil, cenats.Sender{}, fmt.Errorf("failed to create nats protocol: %v", err)
	}

	eClient, err := cloudevents.NewClient(pEvent)
	if err != nil {
		return nil, cenats.Sender{}, fmt.Errorf("failed to create client, %v", err)
	}

	return eClient, *pEvent, nil
}

func newNATSConsumer(natsURL string, natsCredentials nats.Option) (cloudevents.Client, error) {
	pEvent, err := cenats.NewConsumer(natsURL, "mms", cenats.NatsOptions(natsCredentials))
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
		// Silently ignore non product events.
		if !strings.HasPrefix(event.Type(), "no.met.mms.product") {
			return nil
		}

		mmsEvent := ProductEvent{}

		if err := event.DataAs(&mmsEvent); err != nil {
			return fmt.Errorf("failed to decode event as product event: %v", err)
		}

		return callback(&mmsEvent)
	}
}
