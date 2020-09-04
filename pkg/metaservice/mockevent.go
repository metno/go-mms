package metaservice

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/metno/go-mms/pkg/mms"
)

func MockProductEvent(w http.ResponseWriter, r *http.Request) {
	event := cloudevents.NewEvent()

	event.SetID("0173c5ce-e1fb-11ea-9c78-6b708419aa07")
	event.SetSource("ecflow/modellprod")
	event.SetType("no.met.Product.created.v1")
	event.SetSubject("arome.arctic")

	event.SetData(cloudevents.ApplicationJSON, &mms.ProductEvent{
		Product:     "Arome Arctic",
		ProductSlug: "arome.arctic",
		CreatedAt:   time.Now(),
	})

	payload, err := json.Marshal(event)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode event: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	fmt.Fprint(w, string(payload))
}
