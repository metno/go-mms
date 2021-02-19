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
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/metno/go-mms/internal/server"
	"github.com/metno/go-mms/pkg/mms"

	nats "github.com/nats-io/nats-server/v2/server"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

const productionHubName = "default"

func main() {

	var err error
	var hubID string
	var confFile string = "mmsd_config.yml"

	// Create an identifier
	hubID, err = mms.MakeHubIdentifier()
	if err != nil {
		log.Printf("Failed to create identifier, %s", err.Error())
		hubID = "error"
	}

	cmdFlags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "pstorage",
			Aliases: []string{"p"},
			Value:   "./events.sqlite",
			Usage:   "Set persistent event storage location",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "hubid",
			Usage: "Production hub identifier. If not specified, an identifier is generated.",
			Value: hubID,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "hostname",
			Usage: "Specify the hostname for API and NATS.",
			Value: "localhost",
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:  "api-port",
			Usage: "Specify the port number for the API listening port.",
			Value: 8080,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:  "nats-port",
			Usage: "Specify the port number for the NATS listening port.",
			Value: 4222,
		}),
		&cli.StringFlag{
			Name:    "config",
			Aliases: []string{"c"},
			Usage:   "Load configuration from file.",
			EnvVars: []string{"MMSD_CONFIG"},
			Value:   confFile,
		},
	}

	app := &cli.App{
		Before: func(ctx *cli.Context) error {
			inputSource, err := altsrc.NewYamlSourceFromFlagFunc("config")(ctx)
			if err != nil {
				// If there is no file, just return without error
				return nil
			}

			return altsrc.ApplyInputSourceValues(ctx, inputSource, cmdFlags)
		},
		Flags: cmdFlags,
		Action: func(ctx *cli.Context) error {
			natsURL := fmt.Sprintf("nats://%s:%d", ctx.String("hostname"), ctx.Int("nats-port"))
			apiURL := fmt.Sprintf("%s:%d", ctx.String("hostname"), ctx.Int("api-port"))

			natsServer, err := nats.NewServer(&nats.Options{
				ServerName: fmt.Sprintf("mmsd-nats-server-%s", productionHubName),
				Host:       ctx.String("hostname"),
				Port:       ctx.Int("nats-port"),
			})
			if err != nil {
				nats.PrintAndDie(fmt.Sprintf("nats server failed: %s for server: mmsd-nats-server-%s", err, productionHubName))
			}

			cacheDB, err := server.NewDB(ctx.String("pstorage"))
			if err != nil {
				log.Fatalf("could not open cache db: %s", err)
			}

			templates := server.CreateTemplates()
			webService := server.NewService(templates, cacheDB, natsURL)

			startNATSServer(natsServer, natsURL)
			startEventCaching(webService, natsURL)
			startWebServer(webService, apiURL)

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "keygen",
				Usage: "Generate a new API key and add it to the authorized_keys file",
				Flags: []cli.Flag{
					altsrc.NewStringFlag(&cli.StringFlag{
						Name:    "message",
						Aliases: []string{"m"},
						Usage:   "A descriptive message for the key",
					}),
				},
				Action: generateAPIKey(),
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func startNATSServer(natsServer *nats.Server, natsURL string) {
	go func() {
		log.Printf("Starting NATS server on %s ...", natsURL)
		if err := nats.Run(natsServer); err != nil {
			nats.PrintAndDie(err.Error())
		}
		natsServer.WaitForShutdown()
	}()
}

func startEventCaching(webService *server.Service, natsURL string) {
	go func() {
		log.Println("Start caching incoming events ...")

		if err := webService.RunCache(natsURL); err != nil {
			log.Fatalf("Caching events failed: %s", err)
		}
	}()

	// Start a separate go routine for regularly deleting old events from the events cache db.
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := webService.DeleteOldEvents(time.Now().AddDate(0, 0, -3)); err != nil {
					log.Printf("failed to delete old events from cache db: %s", err)
				}
			}
		}
	}()
}

func startWebServer(webService *server.Service, apiURL string) {
	server := &http.Server{
		Addr:         apiURL,
		Handler:      webService.Router,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	log.Printf("Starting webserver on %s ...\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func generateAPIKey() func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		rand.Seed(time.Now().UnixNano())

		byteKey := make([]byte, 32)
		for i := range byteKey {
			byteKey[i] = byte(rand.Intn(256))
		}

		apiKey := base64.StdEncoding.EncodeToString([]byte(byteKey))
		fmt.Println(apiKey)

		return nil
	}
}
