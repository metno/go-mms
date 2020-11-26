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

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/metno/go-mms/pkg/mms"
	"github.com/urfave/cli/v2"
)

func listAllEvents() func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		events := []*mms.ProductEvent{}
		newEvents, err := mms.ListProductEvents(ctx.String("prduction-hub"), mms.Options{})
		if err != nil {
			return fmt.Errorf("failed to access events: %v", err)
		}
		events = append(events, newEvents...)

		for _, event := range events {
			fmt.Printf("Event: %+v\n", event)
		}
		return nil
	}
}

func subscribeEvents() func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		errChannel := make(chan error, 1)
		go func(ctx *cli.Context) {
			mmsClient, err := mms.NewNatsConsumerClient(ctx.String("production-hub"))
			if err != nil {
				errChannel <- err
				return
			}
			mmsClient.WatchProductEvents(productReceiver, mms.Options{})
		}(ctx)
		select {
		case err := <-errChannel:
			return fmt.Errorf("one hub event subscription failed, ending: %v", err)
		}
	}
}

func postEvent() func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		productEvent := mms.ProductEvent{
			JobName:         ctx.String("jobname"),
			Product:         ctx.String("product"),
			ProductLocation: ctx.String("product-location"),
			ProductionHub:   ctx.String("production-hub"),
			CreatedAt:       time.Now(),
			NextEventAt:     time.Now().Add(time.Second * time.Duration(ctx.Int("event-interval"))),
		}

		// hardcoded to test-server. Should be findable from ProductionHub?
		url := ctx.String("production-hub") + "/api/v1/postevent"

		// Create a json-payload from productEvent
		jsonStr, err := json.Marshal(&productEvent)
		// Create a http-request to post the payload
		httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))

		// Hardcoded Api-Key, maybe in productEvent?
		httpReq.Header.Set("Api-Key", "HARDCODED APIKEY CHANGE")
		httpReq.Header.Set("Content-Type", "application/json")

		// Create a http connection to the api.
		httpClient := &http.Client{}
		httpResp, err := httpClient.Do(httpReq)
		if err != nil {
			log.Fatalf("Failed to create http client: %v", err)
		}
		defer httpResp.Body.Close()

		// If 201 is not returned, panic with http response
		if httpResp.StatusCode != http.StatusCreated {
			log.Fatalf("Product event not posted: %s", httpResp.Status)
		}
		return nil

	}
}

func listProductionHubs(ctx *cli.Context) error {
	return nil
}

func productReceiver(event *mms.ProductEvent) error {
	fmt.Println(event)
	return nil
}
