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
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/metno/go-mms/internal/server"
	nats "github.com/nats-io/nats-server/v2/server"
)

func createNats(hostname string, port int) (*nats.Server, error) {
	natsServer, err := nats.NewServer(&nats.Options{
		ServerName: "mmsd-nats-server-test",
		Host:       hostname,
		Port:       port,
	})
	return natsServer, err
}

func createStateDB(t *testing.T) (*sql.DB, string) {
	t.Helper()
	tmpFile, err := os.CreateTemp("", "state_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %s", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	stateDB, err := server.NewStateDB(tmpPath)
	if err != nil {
		t.Fatalf("failed to create state db: %s", err)
	}

	return stateDB, tmpPath
}

// waitForCondition polls the given condition function at the specified interval
// for up to the given timeout duration. Returns true if the condition returns true
// before the timeout, false otherwise.
func waitForCondition(t *testing.T, timeout time.Duration, interval time.Duration, condition func() bool) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

func httpGetWithTimeout(t *testing.T, url string, timeout time.Duration) bool {
	t.Helper()
	started := waitForCondition(t, timeout, 50*time.Millisecond, func() bool {
		resp, err := http.Get(url)
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode > 0
	})
	return started
}


func TestRun(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		expectErr bool
		httpCheck string
	}{
		{
			name:      "valid startup",
			args:      []string{"./mmsd", "-api-port", "1234"},
			expectErr: false,
			httpCheck: "http://localhost:1234/",
		},
		{
			name:      "invalid command",
			args:      []string{"./mmsd", "server_start"},
			expectErr: true,
			httpCheck: "",
		},
		{
			name:      "nonexistent work-dir",
			args:      []string{"./mmsd", "-work-dir", "/nonexistent/path/"},
			expectErr: true,
			httpCheck: "",
		},
		{
			name:      "version command",
			args:      []string{"./mmsd", "version"},
			expectErr: false,
			httpCheck: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = tt.args

			if tt.expectErr {
				err := run()
				if err == nil {
					t.Fatal("expected an error but got nil")
				}
				t.Logf("got expected error: %v", err)
				return
			}

			// For non-error cases, run() blocks so use a goroutine
			errCh := make(chan error, 1)
			go func() {
				errCh <- run()
			}()

			if tt.httpCheck != "" {
				started := httpGetWithTimeout(t, tt.httpCheck, 3*time.Second)
				if !started {
					select {
					case err := <-errCh:
						t.Fatalf("run() returned error: %v", err)
					default:
						t.Fatal("mmsd did not respond to HTTP within timeout")
					}
				}
				t.Log("mmsd is running and responding to HTTP")
			}
		})
	}
}

func TestStartNATSServer(t *testing.T) {
	natsServer, err := createNats("localhost", 4333)
	if err != nil {
		t.Fatalf("failed to create NATS server: %s", err)
	}

	natsURL := "localhost:4333"

	startNATSServer(natsServer, natsURL)
	// Poll the server's Running() method until it returns true or 1 second elapses
	if !waitForCondition(t, 1*time.Second, 10*time.Millisecond, natsServer.Running) {
		t.Fail()
		t.Log("failed to run natsserver")
	}
	natsServer.Shutdown()
}

func TestStartHeartBeat(t *testing.T) {

}

func TestStartEventLoop(t *testing.T) {

}

func TestStartWebServer(t *testing.T) {

}

func TestGenerateAPIKey(t *testing.T) {
	// Create a temporary file for the state database
	stateDB, tmpPath := createStateDB(t)
	defer os.Remove(tmpPath)
	defer stateDB.Close()
	var err error
	// Test generating an API key
	err = generateAPIKey(stateDB, "test key")
	if err != nil {
		t.Fatalf("generateAPIKey returned an error: %s", err)
	}

	// Generate multiple keys and verify they are all added successfully
	for i := range 100 {
		err = generateAPIKey(stateDB, "test key")
		if err != nil {
			t.Fatalf("generateAPIKey returned an error on iteration %d: %s", i, err)
		}
	}

	// Test with an empty message
	err = generateAPIKey(stateDB, "")
	if err != nil {
		t.Fatalf("generateAPIKey returned an error with empty message: %s", err)
	}
}
