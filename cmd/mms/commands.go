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
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/metno/go-mms/pkg/mms"
	"github.com/urfave/cli/v2"
)

func listAllEvents() func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		events := []*mms.ProductEvent{}
		if ctx.String("production-hub") == "" {
			return fmt.Errorf("No production-hub specified")
		}
		url := ctx.String("production-hub") + "/api/v1/events"
		newEvents, err := mms.ListProductEvents(url)
		if err != nil {
			return fmt.Errorf("failed to access events: %v", err)
		}
		events = append(events, newEvents...)

		for _, event := range events {
			fmt.Printf("Event: %+v\n", event)
		}
		return nil
	}
}

func subscribeEvents() func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		errChannel := make(chan error, 1)
		go func(ctx *cli.Context) {
			mmsClient, err := mms.NewNatsConsumerClient(ctx.String("production-hub"))
			if err != nil {
				errChannel <- err
				return
			}
			if ctx.String("command") != "None" {
				callback := createExecutableCallback(ctx.String("command"), ctx.Bool("filter"))
				mmsClient.WatchProductEvents(callback)
			} else {
				// Same as Aviso-echo
				mmsClient.WatchProductEvents(productReceiver)
			}

		}(ctx)
		select {
		case err := <-errChannel:
			return fmt.Errorf("one hub event subscription failed, ending: %v", err)
		}
	}
}

func postEvent() func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		var err error

		refTime := time.Now()
		if ctx.String("reftime") != "now" {
			refTime, err = time.Parse(time.RFC3339, ctx.String("reftime"))
			if err != nil {
				log.Println("Could not parse reftime")
				log.Println("Please use RFC 3339 format:")
				log.Println("- '2006-01-02T15:04:05Z' for UTC")
				log.Println("- '2006-01-02T15:04:05+01:00' for other time zones")
				log.Fatalf("Parser error: %v", err)
			}
		}

		productEvent := mms.ProductEvent{
			JobName:         ctx.String("jobname"),
			Product:         ctx.String("product"),
			ProductLocation: ctx.String("product-location"),
			ProductionHub:   ctx.String("production-hub"),
			Counter:         ctx.Int("counter"),
			TotalCount:      ctx.Int("ntotal"),
			RefTime:         refTime,
			CreatedAt:       time.Now(),
			NextEventAt:     time.Now().Add(time.Second * time.Duration(ctx.Int("event-interval"))),
		}

		if ctx.String("production-hub") == "" {
			return fmt.Errorf("No production-hub specified")
		}
		url := ctx.String("production-hub") + "/api/v1/events"

		// Create a json-payload from productEvent
		jsonStr, err := json.Marshal(&productEvent)
		// Create a http-request to post the payload
		httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))

		httpReq.Header.Set("Api-Key", ctx.String("api-key"))
		httpReq.Header.Set("Content-Type", "application/json")

		// Create a http connection to the api.
		var tr *http.Transport
		if ctx.Bool("insecure") {
			tr = &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
		} else {
			tr = &http.Transport{}
		}

		httpClient := &http.Client{Transport: tr}
		httpResp, err := httpClient.Do(httpReq)
		if err != nil {
			log.Fatalf("Failed to create http client: %v", err)
		}
		defer httpResp.Body.Close()

		// If 201 is not returned, panic with http response
		if httpResp.StatusCode != http.StatusCreated {
			log.Fatalln(httpResp.Status)
		}

		return nil
	}
}

func productReceiver(event *mms.ProductEvent) error {
	fmt.Println(event)
	return nil
}

func createExecutableCallback(filepath string, filter bool) func(event *mms.ProductEvent) error {
	_, err := exec.LookPath(filepath)

	if err != nil {
		log.Print(fmt.Errorf("command executable not found, %s", err.Error()))
		return productReceiver
	}

	return func(event *mms.ProductEvent) error {
		var productLocation string

		if filter {
			productLocation = event.ProductLocation
		} else {
			productLocation = ""
		}
		command := exec.Command(filepath, productLocation)

		var stdout bytes.Buffer
		var stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr

		err = command.Run()

		if err != nil {
			fmt.Println("Failed", err, stderr.String())
			return fmt.Errorf("Failed to run executable, %s", err.Error())
		}

		fmt.Println(stdout.String())
		return nil
	}

}
