package main

import (
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

func main() {
	natsServer, err := nats.NewServer(&nats.Options{
		ServerName: fmt.Sprintf("mmsd-nats-server-%s", productionHubName),
	})
	if err != nil {
		nats.PrintAndDie(fmt.Sprintf("nats server failed: %s for server: mmsd-nats-server-%s", err, productionHubName))
	}

	cacheDB, err := server.NewDB("")
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
