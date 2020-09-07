package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/metno/go-mms/internal/api"
	nats "github.com/nats-io/nats-server/v2/server"
)

const staticFilesDir = "./static/"
const productionHubName = "default"

func main() {
	templates := template.Must(template.ParseGlob("templates/*"))

	service := api.NewService(templates, staticFilesDir)

	log.Println("Starting webserver for internal services ...")
	go func() {
		http.ListenAndServe(":8088", service.InternalRouter)
	}()

	server := &http.Server{
		Addr:         ":8080",
		Handler:      service.ExternalRouter,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	startNATSServer()

	log.Println("Starting webserver ...")
	log.Fatal(server.ListenAndServe())
}

func startNATSServer() {
	s, err := nats.NewServer(&nats.Options{
		ServerName: fmt.Sprintf("mmsd-nats-server-%s", productionHubName),
	})
	if err != nil {
		nats.PrintAndDie(fmt.Sprintf("nats server failed: %s for server: mmsd-nats-server-%s", err, productionHubName))
	}

	go func() {
		log.Println("Starting NATS server...")
		if err := nats.Run(s); err != nil {
			nats.PrintAndDie(err.Error())
		}
		s.WaitForShutdown()
	}()
}
