package cache

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // Import sqlite3 driver for database/sql library
	"github.com/metno/go-mms/pkg/mms"
)

const dbFilePath = "/tmp/eventsCache.db"

// Run starts up a watch of incoming events from NATS. Each incoming event is stored. This function blocks forever.
func Run(nats string) error {
	eventClient, err := mms.NewNatsConsumerClient(nats)
	if err != nil {
		return fmt.Errorf("failed to create nats consumer client: %s", err)
	}

	if err := createCacheDB(dbFilePath); err != nil {
		return err
	}

	sqliteDatabase, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to connect to sqlite database: %s", err)
	}
	defer sqliteDatabase.Close()

	eventClient.WatchProductEvents(cacheProductEventCallback(sqliteDatabase), mms.Options{})

	return nil
}

func cacheProductEventCallback(db *sql.DB) func(e *mms.ProductEvent) error {

	return func(event *mms.ProductEvent) error {
		insertEventSQL := `INSERT INTO event(event) VALUES (?)`
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("failed to create json blob for storage: %s", err)
		}

		statement, err := db.Prepare(insertEventSQL) // Prepare statement.
		_, err = statement.Exec(payload)
		if err != nil {
			return fmt.Errorf("failed to store event in db: %s", err)
		}

		return nil
	}
}

func createCacheDB(dbFilePath string) error {
	file, err := os.Create(dbFilePath) // Create SQLite file
	if err != nil {
		return fmt.Errorf("failed to create db file: %s", err)
	}
	file.Close()

	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return fmt.Errorf("failed to connect to sqlite database: %s", err)
	}
	defer db.Close()

	createEventTableSQL := `CREATE TABLE event (
		"id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,		
		"event" BLOB
	  );` // SQL Statement for Create Table

	statement, err := db.Prepare(createEventTableSQL) // Prepare SQL Statement
	if err != nil {
		return fmt.Errorf("failed to create event table in db: %s", err)
	}
	_, err = statement.Exec() // Execute SQL Statements

	return err
}

// GetAllEvents returns all product events it can find in the cache database.
func GetAllEvents() ([]*mms.ProductEvent, error) {
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite database: %s", err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT * FROM event")
	if err != nil {
		return nil, fmt.Errorf("could not access db to get events: %s", err)
	}
	var events []*mms.ProductEvent
	for rows.Next() {
		var id int
		var payload []byte
		rows.Scan(&id, &payload)

		var event mms.ProductEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			log.Printf("failed to unmarshal an event from db: %s", err)
		} else {
			events = append(events, &event)
		}
	}
	return events, nil
}
