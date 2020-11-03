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
	"fmt"
	"time"

	"github.com/metno/go-mms/pkg/mms"
	"github.com/urfave/cli/v2"
)

func listAllEvents(hubs []mms.ProductionHub) func(*cli.Context) error {
	return func(c *cli.Context) error {
		events := []*mms.ProductEvent{}
		for _, h := range hubs {
			newEvents, err := mms.ListProductEvents(h.EventCache, mms.Options{})
			if err != nil {
				return fmt.Errorf("failed to access events: %v", err)
			}
			events = append(events, newEvents...)
		}

		for _, e := range events {
			fmt.Printf("Event: %+v\n", e)
		}
		return nil
	}
}

func subscribeEvents(hubs []mms.ProductionHub) func(*cli.Context) error {
	return func(c *cli.Context) error {
		errChannel := make(chan error, 1)
		for _, h := range hubs {
			go func(h mms.ProductionHub) {
				mmsClient, err := mms.NewNatsConsumerClient(h.NatsURL)
				if err != nil {
					errChannel <- err
					return
				}
				mmsClient.WatchProductEvents(productReceiver, mms.Options{})
			}(h)

		}
		select {
		case err := <-errChannel:
			return fmt.Errorf("one hub event subscription failed, ending: %v", err)
		}
	}
}

func postEvent(hubs []mms.ProductionHub) func(*cli.Context) error {
	return func(c *cli.Context) error {
		productEvent := mms.ProductEvent{
			Product:       c.String("product"),
			ProductionHub: c.String("production-hub"),
			CreatedAt:     time.Now(),
		}

		return mms.MakeProductEvent(hubs, &productEvent)
	}
}

func listProductionHubs(c *cli.Context) error {
	return nil
}

func productReceiver(e *mms.ProductEvent) error {
	fmt.Println(e)
	return nil
}
