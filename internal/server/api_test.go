package server

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	nats "github.com/nats-io/nats-server/v2/server"
)

var apiKey string = "VnTkRPyNzu3TzeUQhgMmqsh2HEqBLE68MA+3n1yezZA="
var s *Service
var stateDBMock sqlmock.Sqlmock
var eventsDBMock sqlmock.Sqlmock

func TestMain(m *testing.M) {

	s, eventsDBMock, stateDBMock, _ = NewMockService()
	s.setRoutes()

	s.NatsURL = "localhost:4333"

	createStateDBTables(s.stateDB)

	stateDBMock.ExpectPrepare("INSERT INTO api_keys")
	stateDBMock.ExpectExec("INSERT INTO api_keys").WithArgs(apiKey, time.Now().Format(time.RFC3339), "test-key").WillReturnResult(sqlmock.NewResult(1, 1))

	err := AddNewApiKey(s.stateDB, apiKey, "test-key")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	natsServer, err := nats.NewServer(&nats.Options{
		ServerName: "mmsd-nats-server-test",
		Host:       "localhost",
		Port:       4333,
	})

	go func() {
		log.Printf("Starting NATS server ...")
		if err := nats.Run(natsServer); err != nil {
			nats.PrintAndDie(err.Error())
		}
		natsServer.WaitForShutdown()
	}()

	exitCode := m.Run()

	os.Exit(exitCode)
}

func TestRegisterMessage(t *testing.T) {
	payload := []byte(`{"product":"test product"}`)

	stateDBMock.ExpectPrepare("UPDATE api_keys SET lastUsed")
	stateDBMock.ExpectExec("UPDATE api_keys SET lastUsed").WillReturnResult(sqlmock.NewResult(1, 1))
	eventsDBMock.ExpectPrepare("INSERT INTO events").ExpectExec().WillReturnResult(sqlmock.NewResult(1, 1))

	req, _ := http.NewRequest("POST", "/api/v1/events", bytes.NewBuffer(payload))
	req.Header.Add("Api-Key", apiKey)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusCreated, response.Code)
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
