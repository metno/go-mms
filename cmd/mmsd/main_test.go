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
	"os"
	"testing"
	"time"

	nats "github.com/nats-io/nats-server/v2/server"
)

func createNats() (*nats.Server, error) {
	natsServer, err := nats.NewServer(&nats.Options{
		ServerName: "mmsd-nats-server-test",
		Host:       "localhost",
		Port:       4333,
	})
	return natsServer, err
}

func TestMain(t *testing.T) {
	// Define non-failing call (Overwrites test-calls which fails)
	os.Args = []string{"./mmsd"}

	go main()
	for {
		<-time.After(3 * time.Second)
		// overwrite panics from underlying goroutines being abandonded
		// TODO: needs better explanation
		if recover() != nil {
			t.Log("Something has happened")
		}
		t.Log("Did not fail before timeout")
	}

}

func TestStartNATSServer(t *testing.T) {
	natsServer, err := createNats()
	if err != nil {
		t.Fail()
		t.Logf("failed to create NewServer: %s", err)
	}

	natsURL := "localhost:4333"

	if natsServer.Running() {
		t.Fail()
		t.Log("not testing startNATSServer, natsserver already running")
	}

	startNATSServer(natsServer, natsURL)
	// Give go-routine time to start. less hacky way?
	time.Sleep(1 * time.Millisecond)

	if !natsServer.Running() {
		t.Fail()
		t.Log("failed to run natsserver")
	}

}

func TestStartHeartBeat(t *testing.T) {

}

func TestStartEventLoop(t *testing.T) {

}

func TestStartWebServer(t *testing.T) {

}

func TestGenerateAPIKey(t *testing.T) {

}
