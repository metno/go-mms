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
	"net/http"
	"time"

	"github.com/metno/go-mms/pkg/mms"
	"github.com/urfave/cli/v2"
)

func listAllEvents(hubs []mms.ProductionHub) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		events := []*mms.ProductEvent{}
		for _, hub := range hubs {
			newEvents, err := mms.ListProductEvents(hub.EventCache, mms.Options{})
			if err != nil {
				return fmt.Errorf("failed to access events: %v", err)
			}
			events = append(events, newEvents...)
		}

		for _, event := range events {
			fmt.Printf("Event: %+v\n", event)
		}
		return nil
	}
}

func subscribeEvents(hubs []mms.ProductionHub) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		errChannel := make(chan error, 1)
		for _, hub := range hubs {
			go func(hub mms.ProductionHub) {
				mmsClient, err := mms.NewNatsConsumerClient(hub.NatsURL)
				if err != nil {
					errChannel <- err
					return
				}
				mmsClient.WatchProductEvents(productReceiver, mms.Options{})
			}(hub)

		}
		select {
		case err := <-errChannel:
			return fmt.Errorf("one hub event subscription failed, ending: %v", err)
		}
	}
}

func postEvent(hubs []mms.ProductionHub) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		productEvent := mms.ProductEvent{
			JobName:       ctx.String("jobname"),
			Product:       ctx.String("product"),
			ProductionHub: ctx.String("production-hub"),
			CreatedAt:     time.Now(),
			NextEventAt:   time.Now().Add(time.Second * time.Duration(ctx.Int("event-interval"))),
		}

		// hardcoded to test-server. Should be findable from ProductionHub?
		url := "http://mms-test.modellprod.met.no:8080/api/v1/postevent"

		// Create a json-payload from productEvent
		jsonStr, err := json.Marshal(&productEvent)
		// Create a http-request to post the payload
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))

		// Hardcoded Api-Key, maybe in productEvent?
		req.Header.Set("Api-Key", "HARDCODED APIKEY CHANGE")
		req.Header.Set("Content-Type", "application/json")

		// Create a http connection to the api.
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		// If 201 is not returned, panic with http response
		if resp.Status != "201 Created" {
			panic(resp.Status)
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
