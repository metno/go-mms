package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/metno/go-mms/internal/api"
)

const staticFilesDir = "./static/"

func main() {
	templates := template.Must(template.ParseGlob("templates/*"))

	service := api.NewService(templates, staticFilesDir)

	log.Println("Starting webserver...")
	go func() {
		http.ListenAndServe(":8088", service.InternalRouter)
	}()

	server := &http.Server{
		Addr:         ":8080",
		Handler:      service.ExternalRouter,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
