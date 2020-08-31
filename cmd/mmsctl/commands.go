package main

import (
	"fmt"
	"time"

	"github.com/metno/go-mms/pkg/mms"
	"github.com/urfave/cli/v2"
)

type productionHub struct {
	Name       string
	NatsURL    string
	EventCache string
}

func listAllEvents(hubs []productionHub) func(*cli.Context) error {
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

func subscribeEvents(hubs []productionHub) func(*cli.Context) error {
	return func(c *cli.Context) error {
		errChannel := make(chan error, 1)
		for _, h := range hubs {
			go func(h productionHub) {
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

func postEvent(hubs []productionHub) func(*cli.Context) error {
	return func(c *cli.Context) error {
		productEvent := mms.ProductEvent{
			Product:       c.String("product"),
			ProductSlug:   c.String("product-slug"),
			ProductionHub: c.String("production-hub"),
			CreatedAt:     time.Now(),
		}

		var hub productionHub
		for _, h := range hubs {
			if h.Name == c.String("production-hub") {
				hub = h
				break
			}
		}

		if (hub == productionHub{}) {
			return fmt.Errorf("could not find correct hub to send event")
		}

		mmsClient, err := mms.NewNatsSenderClient(hub.NatsURL)
		if err != nil {
			return fmt.Errorf("failed to post event to messaging service: %v", err)
		}

		err = mmsClient.PostProductEvent(&productEvent, mms.Options{})
		if err != nil {
			return fmt.Errorf("failed to post event to messaging service: %v", err)
		}

		return nil
	}
}

func listProductionHubs(c *cli.Context) error {
	return nil
}

func productReceiver(e *mms.ProductEvent) error {
	fmt.Println(e)
	return nil
}
