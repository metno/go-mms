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

func main() {

	var err error
	var hubID string
	var confFile string = "mmsd_config.yml"
	var dbCacheFile string = "events.db"
	var dbStateFile string = "state.db"

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
				log.Fatalf("could not open cache db: %s", err)
			}

			statePath := fmt.Sprint(filepath.Join(ctx.String("work-dir"), dbStateFile))
			stateDB, err := server.NewStateDB(statePath)
			if err != nil {
				log.Fatalf("could not open cache db: %s", err)
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
				Name:  "keygen",
				Usage: fmt.Sprintf("Generate a new API key and add it to the %s file", authKeysFile),
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

func generateAPIKey() func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		rand.Seed(time.Now().UnixNano())

		// Generate the key
		byteKey := make([]byte, 32)
		for i := range byteKey {
			byteKey[i] = byte(rand.Intn(255))
		}

		apiKey := base64.StdEncoding.EncodeToString([]byte(byteKey))

		// Write the File
		outFile, err := os.OpenFile(authKeysFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			panic(err)
		}

		defer outFile.Close()

		keyMsg := ctx.String("message")
		if keyMsg == "" {
			keyMsg = "Unnamed key"
		}
		keyMsg = fmt.Sprintf("%s (%s)", keyMsg, time.Now().Format(time.RFC3339))

		fileEntry := fmt.Sprintf("api-key %s # %s\n", apiKey, keyMsg)
		fmt.Printf("Generated: %s", fileEntry)
		if _, err = outFile.WriteString(fileEntry); err != nil {
			panic(err)
		}

		return nil
	}
}
