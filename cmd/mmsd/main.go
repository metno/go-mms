package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/metno/go-mms/internal/web"
	nats "github.com/nats-io/nats-server/v2/server"
)

const staticFilesDir = "./static/"
const productionHubName = "default"

func main() {
	startNATSServer()
	startWebServer()
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

func startWebServer() {
	templates := template.Must(template.ParseGlob("templates/*"))

	webService := web.NewService(templates, staticFilesDir)

	log.Println("Starting webserver for internal services ...")
	go func() {
		http.ListenAndServe(":8088", webService.InternalRouter)
	}()

	server := &http.Server{
		Addr:         ":8080",
		Handler:      webService.ExternalRouter,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	log.Println("Starting webserver ...")
	log.Fatal(server.ListenAndServe())
}
