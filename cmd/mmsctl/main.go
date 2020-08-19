package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/metno/go-mms/internal/mms"
)

func receive(event cloudevents.Event) {
	fmt.Printf("%s", event)
}

func main() {
	resp, err := http.Get("http://localhost:8080")
	if err != nil {
		log.Fatalf("Could not get events from local http server:%v", err)
	}
	defer resp.Body.Close()

	event := cloudevents.NewEvent()
	if err = json.NewDecoder(resp.Body).Decode(&event); err != nil {
		log.Fatalf("Failed to decode event: %v", err)
	}

	message := mms.METDatasetCreatedMessage{}
	if err = event.DataAs(&message); err != nil {
		log.Fatalf("Failed to decode message in event: %v", err)
	}

	fmt.Printf("Event metadata:\n %+v\n", event.Context)
	fmt.Printf("Message:\n%s\n", printMessage(message))
}

func printMessage(m interface{}) string {
	s, _ := json.MarshalIndent(m, "", "\t")
	return string(s)
}
