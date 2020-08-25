package main

import (
	"fmt"

	"github.com/metno/go-mms/pkg/mms"
	"github.com/urfave/cli/v2"
)

func listAllEvents(c *cli.Context) error {
	events, err := mms.ListDatasetCreatedEvents("http://localhost:8080", mms.Options{})
	if err != nil {
		return fmt.Errorf("Failed to access events: %v", err)
	}

	for _, e := range events {
		fmt.Printf("Event: %+v\n", e)
	}

	return nil
}

func subscribeEvents(c *cli.Context) error {
	err := mms.WatchDatasetCreatedEvents("nats://localhost:4222", mms.Options{}, datasetCreatedReceiver)

	return err
}

func postEvent(c *cli.Context) error {
	return nil
}

func listProductionHubs(c *cli.Context) error {
	return nil
}

func datasetCreatedReceiver(e *mms.DatasetCreatedEvent) error {
	fmt.Println(e)
	return nil
}
