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

const defaultDBFilePath = "/tmp/eventsCache.db"

// NewDB returns an sql database object, initialized with necessary tables.
func NewDB(filePath string) (*sql.DB, error) {
	fp := defaultDBFilePath
	if filePath != "" {
		fp = filePath
	}
	return createCacheDB(fp)
}

// RunCache starts up a watch of incoming events from NATS. Each incoming event is stored in the cache database.
// This function blocks forever, until it fails to subscribe to NATS for some reason.
func (s *Service) RunCache(natsURL string) error {
	eventClient, err := mms.NewNatsConsumerClient(natsURL)
	if err != nil {
		return fmt.Errorf("failed to create nats consumer client: %s", err)
	}

	eventClient.WatchProductEvents(cacheProductEventCallback(s.cacheDB), mms.Options{})

	return nil
}

// GetAllEvents returns all product events it can find in the cache database.
func (s *Service) GetAllEvents(ctx context.Context) ([]*mms.ProductEvent, error) {
	rows, err := s.cacheDB.QueryContext(ctx, "SELECT * FROM events")
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

// DeleteOldEvents removes events older than a specifed datetime.
func (s *Service) DeleteOldEvents(maxAge time.Time) error {
	deleteOldEvents := `DELETE FROM events WHERE createdAt < "` + maxAge.Format(time.RFC3339) + `";`
	_, err := s.cacheDB.Exec(deleteOldEvents)
	return err
}

func cacheProductEventCallback(db *sql.DB) func(e *mms.ProductEvent) error {
	return func(event *mms.ProductEvent) error {
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to create json blob for storage: %s", err)
		}

		insertEventSQL := `INSERT INTO events(createdAt,event) VALUES (?, ?)`
		statement, err := db.Prepare(insertEventSQL)
		_, err = statement.Exec(event.CreatedAt.Format(time.RFC3339), payload)
		if err != nil {
			return fmt.Errorf("failed to store event in db: %s", err)
		}

		return nil
	}
}

func createCacheDB(dbFilePath string) (*sql.DB, error) {
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
