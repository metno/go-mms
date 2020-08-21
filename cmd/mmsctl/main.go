package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/metno/go-mms/internal/mms"
)

func receive(event cloudevents.Event) {
	fmt.Printf("%s", event)
}

func main() {
	app := &cli.App{
		Name:  "mmsctl",
		Usage: "Get and post events and get info about production hubs, by talking to the MET Messaging system",
		Commands: []*cli.Command{
			{
				Name:    "list-all",
				Aliases: []string{"ls"},
				Usage:   "List all the latest available events in the system",
				Action:  listAllEvents,
			},
			{
				Name:    "subscribe",
				Aliases: []string{"s"},
				Usage:   "Listen for new incoming events, get them printed continuously. Optionally, set up filters to limit events you get.",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "source", Usage: "Filter incoming events by setting specifying the source events are coming from.", Aliases: []string{"s"}},
				},
				Action: subscribeEvents,
			},
			{
				Name:    "post",
				Aliases: []string{"p"},
				Usage:   "Post a message about a product update.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "type",
						Usage: "Type of event. Default is created, but you can set the following type: created, updated, deleted.",
						Value: "created",
					},
				},
				Action: postEvent,
			},
			{
				Name:   "list-production-hubs",
				Usage:  "List all available production hubs, aka. sources of events.",
				Action: listProductionHubs,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

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

	message := mms.METDatasetCreatedMessage{}
	if err = event.DataAs(&message); err != nil {
		log.Fatalf("Failed to decode message in event: %v", err)
	}

	fmt.Printf("Event metadata:\n %+v\n", event.Context)
	fmt.Printf("Message:\n%s\n", printMessage(message))

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

func printMessage(m interface{}) string {
	s, _ := json.MarshalIndent(m, "", "\t")
	return string(s)
}
