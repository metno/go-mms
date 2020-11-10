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
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/metno/go-mms/internal/server"
	"github.com/metno/go-mms/pkg/mms"
	nats "github.com/nats-io/nats-server/v2/server"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

const staticFilesDir = "./static/"
const productionHubName = "default"

func main() {

	// Default file name for config
	// Could be expanded to check and pick a file from a pre-defined list
	var confFile string = "mmsd_config.yml"

	// Create an identifier
	hubID, idErr := mms.MakeHubIdentifier()
	log.Print(hubID)
	if idErr != nil {
		log.Printf("Failed to create identifier, %s", idErr.Error())
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
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:  "port",
			Usage: "Specify the port number for the API lisetning port.",
			Value: 8080,
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
		Action: func(c *cli.Context) error {
			natsServer, err := nats.NewServer(&nats.Options{
				ServerName: fmt.Sprintf("mmsd-nats-server-%s", productionHubName),
			})
			if err != nil {
				nats.PrintAndDie(fmt.Sprintf("nats server failed: %s for server: mmsd-nats-server-%s", err, productionHubName))
			}

			cacheDB, err := server.NewDB(c.String("pstorage"))
			if err != nil {
				log.Fatalf("could not open cache db: %s", err)
			}
			templates := template.Must(template.ParseGlob("templates/*"))
			webService := server.NewService(templates, staticFilesDir, cacheDB)

			startNATSServer(natsServer)
			startEventCaching(webService, "nats://localhost:4222")
			startWebServer(webService)

			return nil
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func startNATSServer(s *nats.Server) {
	go func() {
		log.Println("Starting NATS server on localhost:4222...")
		if err := nats.Run(s); err != nil {
			nats.PrintAndDie(err.Error())
		}
		s.WaitForShutdown()
	}()
}

func startEventCaching(webService *server.Service, natsURL string) {
	go func() {
		log.Println("Start caching incoming events...")

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

func startWebServer(webService *server.Service) {
	server := &http.Server{
		Addr:         ":8080",
		Handler:      webService.Router,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	log.Printf("Starting webserver on %s...\n", server.Addr)
	log.Fatal(server.ListenAndServe())
}
