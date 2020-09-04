package main

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/metno/go-mms/internal/greetings"
)

const staticFilesDir = "./static/"

func main() {
	templates := template.Must(template.ParseGlob("templates/*"))

	greetingsService := greetings.NewService(templates, staticFilesDir)

	log.Println("Starting webserver...")
	go func() {
		http.ListenAndServe(":8088", greetingsService.InternalRouter)
	}()

	server := &http.Server{
		Addr:         ":8080",
		Handler:      greetingsService.ExternalRouter,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
