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

const staticFilesDir = "./../../static/"
const productionHubName = "default"

func TestGetAllEvents(t *testing.T) {
	service, mock, err := NewMockService()
	if err != nil {
		t.Errorf("failed to setup mock service: %s", err)
	}
	// Add expected queries and results to the mock sqlite db.
	mock.ExpectQuery("SELECT (.+) FROM events").
		WillReturnRows(sqlmock.NewRows([]string{"id", "event"}).
			AddRow(1, `{
				"ProductionHub": "ecflow.modellprod",
				"ProductSlug": "arome.arctic",
				"CreatedAt": "2020-08-26T12:18:48.281847242+02:00",
				"ProductLocation": ""
				}`))

	events, err := service.GetAllEvents(context.Background())
	if err != nil {
		t.Errorf("failed to get events from mock service db: %s", err)
	}

	if len(events) != 1 {
		t.Errorf("Expect 1 events; Got %d events", len(events))
	}
}

func TestNewDB(t *testing.T) {
	dbTestFile := fmt.Sprintf("/tmp/mmsdtestsqlite%d.db", rand.Int())
	db, err := NewDB(dbTestFile)
	if err != nil {
		t.Errorf("failed to create db: %s", err)
	}

	var name string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='events'").Scan(&name)
	if err != nil {
		t.Errorf("Expected: events table in db; Got: no events table in db: %s", err)
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
