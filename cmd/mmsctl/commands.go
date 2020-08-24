package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/metno/go-mms/internal/mms"
	"github.com/urfave/cli/v2"
)

func listAllEvents(c *cli.Context) error {
	resp, err := http.Get("http://localhost:8080")
	if err != nil {
		log.Fatalf("Could not get events from local http server:%v", err)
	}
	defer resp.Body.Close()

	event := cloudevents.NewEvent()
	if err = json.NewDecoder(resp.Body).Decode(&event); err != nil {
		log.Fatalf("Failed to decode event: %v", err)
	}

	eventData := mms.METDatasetCreatedEvent{}
	if err = event.DataAs(&eventData); err != nil {
		log.Fatalf("Failed to decode message in event: %v", err)
	}

	fmt.Printf("Event metadata:\n %+v\n", event.Context)
	fmt.Printf("Message:\n%s\n", _printEvent(eventData))

	return nil
}

func subscribeEvents(c *cli.Context) error {
	time.Sleep(10 * time.Second)
	return nil
}

func postEvent(c *cli.Context) error {
	return nil
}

func listProductionHubs(c *cli.Context) error {
	return nil
}

func _printEvent(m interface{}) string {
	s, _ := json.MarshalIndent(m, "", "\t")
	return string(s)
}
