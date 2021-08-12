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
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

// NewStateDB returns an sql database object, initialised with necessary tables.
func NewStateDB(filePath string) (*sql.DB, error) {
	if filePath == "" {
		return nil, fmt.Errorf("empty file path for sqlite database")
	}
	return createStateDB(filePath)
}

// AddNewApiKey adds a given key and message to the keys table. Invalid keys are rejected.
func AddNewApiKey(db *sql.DB, apiKey string, keyMsg string) error {
	err := checkKeyFormat(apiKey)
	if err != nil {
		return fmt.Errorf("api key rejected: %s", err)
	}

	// Insert it into the database. Duplicate entries will be rejected.
	insertSQL := `INSERT INTO api_keys (apiKey, createdDate, createMsg) VALUES (?, ?, ?)`
	statement, err := db.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %s", err)
	}
	_, err = statement.Exec(apiKey, time.Now().Format(time.RFC3339), keyMsg)
	if err != nil {
		return fmt.Errorf("failed to add api key to db: %s", err)
	}

	return nil
}

// RemoveApiKey removes a given key from the keys table. Invalid keys are rejected.
func RemoveApiKey(db *sql.DB, apiKey string) (bool, error) {
	err := checkKeyFormat(apiKey)
	if err != nil {
		return false, fmt.Errorf("api key rejected: %s", err)
	}

	// Delete the key from the database
	deleteSQL := `DELETE FROM api_keys WHERE apiKey = ?`
	statement, err := db.Prepare(deleteSQL)
	if err != nil {
		return false, fmt.Errorf("failed to prepare statement: %s", err)
	}
	result, err := statement.Exec(apiKey)
	if err != nil {
		return false, fmt.Errorf("failed to remove api key from db: %s", err)
	}
	nRows, err := result.RowsAffected()

	return nRows == 1, err
}

// ValidateApiKey checks a given key against the keys table. Invalid keys are rejected.
func ValidateApiKey(db *sql.DB, apiKey string) (bool, error) {
	err := checkKeyFormat(apiKey)
	if err != nil {
		return false, fmt.Errorf("api key rejected: %s", err)
	}

	checkSQL := `UPDATE api_keys SET lastUsed = ? WHERE apiKey = ?`
	statement, err := db.Prepare(checkSQL)
	if err != nil {
		return false, fmt.Errorf("failed to prepare statement: %s", err)
	}
	result, err := statement.Exec(time.Now().Format(time.RFC3339), apiKey)
	if err != nil {
		return false, fmt.Errorf("failed to update api key record in db: %s", err)
	}
	nRows, err := result.RowsAffected()

	return nRows == 1, err
}

// ListApiKeys lists all keys in the keys table
func ListApiKeys(db *sql.DB) error {
	result, err := db.Query("SELECT apiKey, createdDate, lastUsed, createMsg FROM api_keys ORDER BY createdDate ASC")
	if err != nil {
		return fmt.Errorf("failed to list api keys from db: %s", err)
	}
	defer result.Close()

	fmt.Printf("%-44s  %-25s  %-25s  %s\n", "API Key", "Created On", "Last Used", "Message")
	for result.Next() {
		var apiKey string
		var createdDate string
		var lastUsedNull sql.NullString
		var lastUsed string
		var createMsg string
		result.Scan(&apiKey, &createdDate, &lastUsedNull, &createMsg)
		if lastUsedNull.Valid {
			lastUsed = lastUsedNull.String
		} else {
			lastUsed = "Never Used"
		}
		fmt.Printf("%-44s  %-25s  %-25s  %s\n", apiKey, createdDate, lastUsed, createMsg)
	}

	return nil
}

// Check that the key is a base64 encoded 32 byte string
func checkKeyFormat(apiKey string) error {
	rawKey, err := base64.StdEncoding.DecodeString(apiKey)
	if err != nil {
		return fmt.Errorf("the key could not be base64 decoded: %s", err)
	}
	if len(rawKey) != 32 {
		return fmt.Errorf("the key is not 256 bit")
	}
	return nil
}

func createStateDB(dbFilePath string) (*sql.DB, error) {
	// Create database file if it does not exist.
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
	createTable := `CREATE TABLE IF NOT EXISTS "api_keys" (
		"apiKey" TEXT UNIQUE,
		"createdDate" TEXT,
		"lastUsed" TEXT,
		"createMsg" TEXT,
		PRIMARY KEY("apiKey")
	);
	CREATE INDEX IF NOT EXISTS "api_keys_idx" ON "api_keys" (
		"apiKey"
	);`

	_, err = db.Exec(createTable)

	return db, err
}
