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

package server

import (
	"context"
	"fmt"
	"html/template"
	"math/rand"
	"os"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// FIXME: (20200910) We need a better way to initialize the service when testing, rather than having to reference these static files through relative paths like this.
const staticFilesDir = "./../../static/"
const productionHubName = "default"

func TestGetAllEvents(tT *testing.T) {
	service, mock, err := NewMockService()
	if err != nil {
		tT.Errorf("failed to setup mock service: %s", err)
	}
	// Add expected queries and results to the mock sqlite db.
	mock.ExpectQuery("SELECT (.+) FROM events").
		WillReturnRows(sqlmock.NewRows([]string{"id", "createdAt", "event"}).
			AddRow(1, "2020-08-26T12:18:48.281847242+02:00", `{
				"ProductionHub": "ecflow.modellprod",
				"Product": "arome_arctic_sfx_2_5km",
				"CreatedAt": "2020-08-26T12:18:48.281847242+02:00",
				"ProductLocation": ""
				}`))

	events, err := service.GetAllEvents(context.Background())
	if err != nil {
		tT.Errorf("failed to get events from mock service db: %s", err)
	}

	if len(events) != 1 {
		tT.Errorf("Expect 1 events; Got %d events", len(events))
	}
}

func TestNewDB(tT *testing.T) {
	dbTestFile := fmt.Sprintf("/tmp/mmsdtestsqlite%d.db", rand.Int())
	db, err := NewDB(dbTestFile)
	if err != nil {
		tT.Errorf("failed to create db: %s", err)
	}

	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='events'").Scan(&name)
	if err != nil {
		tT.Errorf("Expected: events table in db; Got: no events table in db: %s", err)
	}
	os.Remove(dbTestFile)
}

func NewMockService() (*Service, sqlmock.Sqlmock, error) {
	cacheDB, mock, err := sqlmock.New()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create mock cache DB: %s", err)
	}

	templates := template.Must(template.ParseGlob("./../../templates/*"))
	webService := NewService(templates, staticFilesDir, cacheDB)

	return webService, mock, nil
}
