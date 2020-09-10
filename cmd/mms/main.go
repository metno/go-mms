package main

import (
	"log"
	"os"

	"github.com/metno/go-mms/pkg/mms"
	"github.com/urfave/cli/v2"
)

func main() {

	// Get ProductionHubs to contact
	hubs := mms.ListProductionHubs()

	app := &cli.App{
		Name:  "mms",
		Usage: "Get and post events and get info about production hubs, by talking to the MET Messaging system",
		Commands: []*cli.Command{
			{
				Name:    "list-all",
				Aliases: []string{"ls"},
				Usage:   "List all the latest available events in the system",
				Action:  listAllEvents(hubs),
			},
			{
				Name:    "subscribe",
				Aliases: []string{"s"},
				Usage:   "Listen for new incoming events, get them printed continuously. Optionally, set up filters to limit events you get.",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "source", Usage: "Filter incoming events by setting specifying the source events are coming from.", Aliases: []string{"s"}},
				},
				Action: subscribeEvents(hubs),
			},
			{
				Name:    "post",
				Aliases: []string{"p"},
				Usage:   "Post a message about a product update.",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "production-hub",
						Usage: "Name of the production-hub",
					},
					&cli.StringFlag{
						Name:  "product",
						Usage: "Name of the product.",
					},
					&cli.StringFlag{
						Name:  "type",
						Usage: "Type of event. Default is created, but you can set the following type: created, updated, deleted.",
						Value: "created",
					},
				},
				Action: postEvent(hubs),
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
