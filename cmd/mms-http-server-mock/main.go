package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/metno/go-mms/internal/mms"
)

func main() {
	http.HandleFunc("/", mockDatasetEvent)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func mockDatasetEvent(w http.ResponseWriter, r *http.Request) {
	event := cloudevents.NewEvent()

	event.SetID("0173c5ce-e1fb-11ea-9c78-6b708419aa07")
	event.SetSource("ecflow/modellprod")
	event.SetType("no.met.dataset.created.v1")
	event.SetData(cloudevents.ApplicationJSON, &mms.METDatasetCreatedEvent{
		Name:          "Arome Arctic",
		ReferenceTime: "2020-08-17T00:00:00Z",
	})

	payload, err := json.Marshal(event)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode event: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	fmt.Fprint(w, string(payload))
}
