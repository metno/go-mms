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
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/metno/go-mms/internal/server"
	nats "github.com/nats-io/nats-server/v2/server"
)

const staticFilesDir = "./static/"
const productionHubName = "default"

var persistentStorageLocation = flag.String("p", "./events.sqlite", "Set persistent event storage location.")

func main() {
	flag.Parse()
	natsServer, err := nats.NewServer(&nats.Options{
		ServerName: fmt.Sprintf("mmsd-nats-server-%s", productionHubName),
	})
	if err != nil {
		nats.PrintAndDie(fmt.Sprintf("nats server failed: %s for server: mmsd-nats-server-%s", err, productionHubName))
	}

	cacheDB, err := server.NewDB(*persistentStorageLocation)
	if err != nil {
		log.Fatalf("could not open cache db: %s", err)
	}
	templates := template.Must(template.ParseGlob("templates/*"))
	webService := server.NewService(templates, staticFilesDir, cacheDB)

	startNATSServer(natsServer)
	startEventCaching(webService, "nats://localhost:4222")
	startWebServer(webService)
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
