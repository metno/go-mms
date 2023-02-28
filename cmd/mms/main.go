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

func run(args []string) error {
	// Default file name for config
	// Could be expanded to check and pick a file from a pre-defined list
	var confFile string = "mms_config.yml"

	listFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  "production-hub", // HTTP
			Usage: "The production hub URL.",
		},
	}

	subscriptionFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  "production-hub", // NATS
			Usage: "The production hub NATS URL.",
		},
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "command",
			Usage:   "File location of script or executable run after incoming event.",
			Value:   "None",
			Aliases: []string{"cmd"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "cred-file",
			Usage: "File-path to credfile for subscribing",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "product",
			Usage:   "Name of the product.",
			EnvVars: []string{"MMS_PRODUCT"},
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:  "args",
			Usage: "Toggles sending of productLocation as arg[1] in executable",
			Value: true,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "queue-name",
			Usage:   "Name of NATS subject (queue) to subscribe to",
			EnvVars: []string{"MMS_QUEUE"},
		}),
	}

	postFlags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "api-key",
			Usage:   "The authorized API key.",
			EnvVars: []string{"MMS_API_KEY"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "production-hub", // HTTP
			Usage:   "The production hub URL.",
			EnvVars: []string{"MMS_PRODUCTION_HUB"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "queue-name",
			Usage:   "Nats queue (Subject) to post to.",
			EnvVars: []string{"MMS_QUEUE"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "product",
			Usage:   "Name of the product.",
			EnvVars: []string{"MMS_PRODUCT"},
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "MMD", // xml
			Usage:   "The MMD file describing the product.",
			EnvVars: []string{"MMS_MMD"},
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
			Usage: "The event reference time in RFC3339 format ('2006-01-02T15:04:05Z' or '2006-01-02T15:04:05+01:00').",
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
			Usage:   "A counter value for when the event is one of a series of connected events.",
			Value:   1,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:    "ntotal",
			Aliases: []string{"n"},
			Usage:   "The total number of events to expect for this series of connected events.",
			Value:   1,
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:  "insecure",
			Usage: "Accept invalid certificates, useful for testing with self-signed ones.",
			Value: false,
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
				Usage:   "List all the latest available events in the system.",
				Flags:   listFlags,
				Action:  listAllEventsCmd,
			},
			{
				Name:    "subscribe",
				Aliases: []string{"s"},
				Usage:   "Listen for new incoming events, get them printed continuously.",
				Flags:   subscriptionFlags,
				Action:  subscribeEventsCmd,
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
				Action: postEventCmd,
			},
		},
	}

	return app.Run(args)
}

func main() {
	err := run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
