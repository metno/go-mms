/*
Copyright 2020–2021 MET Norway

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
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

func main() {

	// Default file name for config
	// Could be expanded to check and pick a file from a pre-defined list
	var confFile string = "mms_config.yml"

	listFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  "production-hub", // HTTP
			Usage: "The production hub HTTP URL",
		},
	}

	subscriptionFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  "production-hub", // NATS
			Usage: "The production hub NATS URL",
		},
	}

	postFlags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "api-key",
			Usage: "The authorized API key",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "production-hub", // HTTP
			Usage:   "The production hub HTTP URL",
			EnvVars: []string{"MMS_PRODUCTION_HUB"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "product",
			Usage:   "Name of the product.",
			EnvVars: []string{"MMS_PRODUCT"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "product-location",
			Usage:   "Location of the product.",
			EnvVars: []string{"MMS_PRODUCT_LOCATION"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "jobname",
			Usage:   "Name of the job.",
			EnvVars: []string{"MMS_JOBNAME"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "reftime",
			Usage: "The event reference time.",
			Value: "now",
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:    "event-interval",
			Usage:   "Expected time between events (in seconds).",
			EnvVars: []string{"MMS_EVENT_INTERVAL"},
			Value:   0,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:    "counter",
			Aliases: []string{"i"},
			Usage:   "A counter value if the event is one of a series of connected events.",
			Value:   1,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:    "ntotal",
			Aliases: []string{"n"},
			Usage:   "The total number of events to expect for this series.",
			Value:   1,
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:  "insecure",
			Usage: "Accept invalid certificates, useful for testing with self-signed ones.",
			Value: false,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "type",
			Usage: "Type of event. Default is created, but you can set the following type: created, updated, deleted.",
			Value: "created",
		}),
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Load configuration from file.",
			EnvVars: []string{"MMS_CONFIG"},
			Value:   confFile,
		},
	}

	app := &cli.App{
		Name:  "mms",
		Usage: "Get and post events by talking to the MET Messaging System",
		Commands: []*cli.Command{
			{
				Name:    "list-all",
				Aliases: []string{"ls"},
				Usage:   "List all the latest available events in the system",
				Flags:   listFlags,
				Action:  listAllEvents(),
			},
			{
				Name:    "subscribe",
				Aliases: []string{"s"},
				Usage:   "Listen for new incoming events, get them printed continuously. Optionally, set up filters to limit events you get.",
				Flags:   subscriptionFlags,
				Action:  subscribeEvents(),
			},
			{
				Name:    "post",
				Aliases: []string{"p"},
				Usage:   "Post a message about a product update.",
				Before: func(ctx *cli.Context) error {
					inputSource, err := altsrc.NewYamlSourceFromFile(ctx.String("config"))
					if err != nil {
						// If there is no file, just return without error
						return nil
					}

					return altsrc.ApplyInputSourceValues(ctx, inputSource, postFlags)
				},
				Flags:  postFlags,
				Action: postEvent(),
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
