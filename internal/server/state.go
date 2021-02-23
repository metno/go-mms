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
	"fmt"
	"os"
)

// NewCacheDB returns an sql database object, initialized with necessary tables.
func NewStateDB(filePath string) (*sql.DB, error) {
	if filePath == "" {
		return nil, fmt.Errorf("empty file path for sqlite database")
	}
	return createStateDB(filePath)
}

func createStateDB(dbFilePath string) (*sql.DB, error) {
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
	createTable := `CREATE TABLE IF NOT EXISTS "api_keys" (
		"apiKey" TEXT UNIQUE,
		"createdDate" TEXT,
		"lastUsed" TEXT,
		"createMsg" TEXT,
		PRIMARY KEY("apiKey")
	);`

	_, err = db.Exec(createTable)

	return db, err
}
