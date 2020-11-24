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

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/metno/go-mms/pkg/mms"
)

func MockProductEvent(httpRespW http.ResponseWriter, httpReq *http.Request) {
	event := cloudevents.NewEvent()

	event.SetID("0173c5ce-e1fb-11ea-9c78-6b708419aa07")
	event.SetSource("ecflow/modellprod")
	event.SetType("no.met.Product.created.v1")
	event.SetSubject("arome.arctic")

	event.SetData(cloudevents.ApplicationJSON, &mms.ProductEvent{
		Product:   "arome_arctic_sfx_2_5km",
		CreatedAt: time.Now(),
	})

	payload, err := json.Marshal(event)
	if err != nil {
		http.Error(httpRespW, fmt.Sprintf("Failed to encode event: %v", err), http.StatusInternalServerError)
		return
	}
	httpRespW.Header().Add("Content-Type", "application/json")
	fmt.Fprint(httpRespW, string(payload))
}
