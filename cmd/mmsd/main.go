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
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/metno/go-mms/internal/server"
	"github.com/metno/go-mms/pkg/mms"

	nats "github.com/nats-io/nats-server/v2/server"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

const productionHubName = "default"
const authKeysFile = "mms_authorized_keys"
const confFile = "mmsd_config.yml"
const dbCacheFile = "events.db"
const dbStateFile = "state.db"

func main() {

	var err error
	var hubID string

	// Create an identifier
	hubID, err = mms.MakeHubIdentifier()
	if err != nil {
		log.Printf("Failed to create identifier, %s", err.Error())
		hubID = "error"
	}

	cmdFlags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "work-dir",
			Aliases: []string{"w"},
			Value:   ".",
			Usage:   "The working directory where the files for this instance are stored.",
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
	}

	app := &cli.App{
		Before: func(ctx *cli.Context) error {
			confPath := fmt.Sprint(filepath.Join(ctx.String("work-dir"), confFile))
			inputSource, err := altsrc.NewYamlSourceFromFile(confPath)
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

			cachePath := fmt.Sprint(filepath.Join(ctx.String("work-dir"), dbCacheFile))
			cacheDB, err := server.NewCacheDB(cachePath)
			if err != nil {
				log.Fatalf("could not open events db: %s", err)
			}

			statePath := fmt.Sprint(filepath.Join(ctx.String("work-dir"), dbStateFile))
			stateDB, err := server.NewStateDB(statePath)
			if err != nil {
				log.Fatalf("could not open state db: %s", err)
			}

			templates := server.CreateTemplates()
			webService := server.NewService(templates, cacheDB, stateDB, natsURL)

			startNATSServer(natsServer, natsURL)
			startEventHistoryPurger(webService)
			startWebServer(webService, apiURL)

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "keys",
				Usage: fmt.Sprintf("Manage API keys."),
				Flags: []cli.Flag{
					altsrc.NewBoolFlag(&cli.BoolFlag{
						Name:    "gen",
						Aliases: []string{"g"},
						Usage:   "Generate a new API key and add it to the autorized keys.",
					}),
					altsrc.NewStringFlag(&cli.StringFlag{
						Name:    "message",
						Aliases: []string{"m"},
						Usage:   "A descriptive message for the key.",
						Value:   "Unnamed key",
					}),
				},
				Action: func(ctx *cli.Context) error {
					// Open the database
					statePath := fmt.Sprint(filepath.Join(ctx.String("work-dir"), dbStateFile))
					stateDB, err := server.NewStateDB(statePath)
					if err != nil {
						log.Fatalf("could not open state db: %s", err)
					}

					if ctx.Bool("generate") {
						err := generateAPIKey(stateDB, ctx.String("message"))
						if err != nil {
							log.Fatalf("key generation failed: %s", err)
						}
					}

					return nil
				},
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

func startEventHistoryPurger(webService *server.Service) {
	log.Printf("Starting event history purger...")
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

func generateAPIKey(stateDB *sql.DB, keyMsg string) error {
	// Seeding the random generator for each call may be risky since it may produce the same
	// seed twice if the time resolution is low and the function is called often. However, the
	// function is only called once in a single instance of mmsd, and the database should error
	// on a duplicate key entry.
	rand.Seed(time.Now().UnixNano())

	// Generate the key
	byteKey := make([]byte, 32)
	for i := range byteKey {
		byteKey[i] = byte(rand.Intn(255))
	}
	apiKey := base64.StdEncoding.EncodeToString([]byte(byteKey))

	// Save the new key entry
	err := server.AddNewApiKey(stateDB, apiKey, keyMsg)
	if err != nil {
		log.Fatalf("error in state db: %s", err)
	}

	log.Printf("Generated Key: %s\n", apiKey)

	return nil
}
