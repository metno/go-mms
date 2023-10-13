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
	"bufio"
	"database/sql"
	"fmt"
	"os"
)

// NewStateDB returns an sql database object, initialised with necessary tables.
func NewJWTDB(filePath string, NSC_creds_location string) (*sql.DB, error) {
	if filePath == "" {
		return nil, fmt.Errorf("empty file path for sqlite database")
	}
	return createJWTDB(filePath, NSC_creds_location)
}

func createJWTDB(dbFilePath string, NSC_creds_location string) (*sql.DB, error) {
	// Create database file if it does not exist.
	file, err := os.OpenFile(dbFilePath, os.O_TRUNC, 0660)
	if err != nil {
		return nil, fmt.Errorf("failed to create db file: %s", err)
	}
	file.Close()

	// Create db with necessary tables if it does not exist.
	db, err := sql.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to sqlite database: %s", err)
	}
	createTable := `CREATE TABLE IF NOT EXISTS "jwt_keys" (
		"JWTKey" TEXT UNIQUE,
		"NSC_cred_path" TEXT UNIQUE,
		"lastUsed" TEXT,
		PRIMARY KEY("JWTKey")
	);
	CREATE INDEX IF NOT EXISTS "JWT_keys_idx" ON "jwt_keys" (
		"JWTKey"
	);`

	_, err = db.Exec(createTable)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %s", err)
	}
	err = InitializeJWTDB(db, NSC_creds_location)
	return db, err
}

func ValidateJWTKey(db *sql.DB, JWTKey string) (bool, string, error) {
	var natsUser string
	readSQL := `SELECT NSC_cred_path FROM jwt_keys WHERE JWTKey = ?`
	statement, err := db.Prepare(readSQL)
	err = statement.QueryRow(JWTKey).Scan(&natsUser)
	if err != nil {
		return false, "", fmt.Errorf("failed to retrieve NSC_cred_path record in db: %s", err)
	}
	err = statement.Close()

	return true, natsUser, err
}

func InitializeJWTDB(db *sql.DB, NSC_creds_location string) error {
	var JWTKey string
	error_string := ""
	// Iterate over JWT-locations and NSC locations on startup
	files, err := os.ReadDir(NSC_creds_location)
	if err != nil {
		return fmt.Errorf("failed to list Credfiles at %s to get JWT-tokens: %s", NSC_creds_location, err)
	}
	for _, NSC_cred_path_local := range files {
		// Split JWTKEY to get key (filename in nsc) NO
		// Read JWTKEY from NSC_cred, assume line 1: BEGIN NATS JWT, line 2: JWT, line 3: END NATS JWT
		NSC_cred_path := NSC_creds_location + NSC_cred_path_local.Name()
		f, err := os.Open(NSC_cred_path)
		if err != nil {
			error_string += fmt.Sprintf("failed to open file to find JWT: %s\n", err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		JWTKey = ""
		line := 0
		for scanner.Scan() {
			if line == 1 {
				JWTKey = scanner.Text()
				break
			}
			line++
		}
		if JWTKey == "" {
			error_string += fmt.Sprintf("failed to find JWTK key in file: %s\n", NSC_cred_path)
		} else {
			// Insert it into the database. Duplicate entries will be rejected.
			insertSQL := `INSERT INTO jwt_keys (JWTKey, NSC_cred_path) VALUES (?, ?)`
			statement, err := db.Prepare(insertSQL)
			if err != nil {
				error_string += fmt.Sprintf("failed to prepare insert: %s", err)
			} else {
				_, err = statement.Exec(JWTKey, NSC_cred_path)
				if err != nil {
					error_string += fmt.Sprintf("failed to add JWT key to db: %s ", err)
					error_string += fmt.Sprintf("key: %s, path: %s.", JWTKey, NSC_cred_path)
				}
			}

		}
	}
	if error_string != "" {
		return fmt.Errorf(error_string)
	}
	return nil
}
