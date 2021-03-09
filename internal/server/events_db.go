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

package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3" // Import sqlite3 driver for database/sql library
	"github.com/metno/go-mms/pkg/mms"
)

// NewEventsDB returns an sql database object, initialized with necessary tables.
func NewEventsDB(filePath string) (*sql.DB, error) {
	if filePath == "" {
		return nil, fmt.Errorf("empty file path for sqlite database")
	}
	return createEventsDB(filePath)
}

// GetAllEvents returns all product events in the events database.
func (service *Service) GetAllEvents(ctx context.Context) ([]*mms.ProductEvent, error) {
	rows, err := service.eventsDB.QueryContext(ctx, "SELECT * FROM events")
	if err != nil {
		return nil, fmt.Errorf("could not access db to get events: %s", err)
	}
	defer rows.Close()

	var events []*mms.ProductEvent
	for rows.Next() {
		var id int
		var createdAt string
		var payload []byte
		rows.Scan(&id, &createdAt, &payload)

		var event mms.ProductEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Printf("failed to unmarshal an event from db: %s", err)
		} else {
			events = append(events, &event)
		}
	}
	return events, nil
}

// DeleteOldEvents removes events older than a specified datetime.
func (service *Service) DeleteOldEvents(maxAge time.Time) error {
	deleteOldEvents := `DELETE FROM events WHERE createdAt < "` + maxAge.Format(time.RFC3339) + `";`
	_, err := service.eventsDB.Exec(deleteOldEvents)
	return err
}

func saveProductEvent(db *sql.DB, event *mms.ProductEvent) error {

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to create json blob for storage: %s", err)
	}

	insertEventSQL := `INSERT INTO events(createdAt,event) VALUES (?, ?)`
	statement, err := db.Prepare(insertEventSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %s", err)
	}
	_, err = statement.Exec(event.CreatedAt.Format(time.RFC3339), payload)
	if err != nil {

		return fmt.Errorf("failed to store event in db: %s", err)
	}

	_ = statement.Close()
	return nil
}

func createEventsDB(dbFilePath string) (*sql.DB, error) {
	// Create file if it does not exist.
	file, err := os.OpenFile(dbFilePath, os.O_CREATE, 0660)
	if err != nil {
		return nil, fmt.Errorf("failed to create db file: %s", err)
	}
	file.Close()

	// Create db with necessary tables if it does not exist.
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite database: %s", err)
	}
	createEventTable := `CREATE TABLE IF NOT EXISTS events (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
		"createdAt" string,
		"event" BLOB
	);`

	_, err = db.Exec(createEventTable)

	return db, err
}
