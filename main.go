package main

import (
	"database/sql"
	"fmt"
	"os"
	"sync"

	"github.com/dustin/go-humanize"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

type Row struct {
	Id             int64
	BigIntColumn   uint64
	TimestampStart int
	TimestampEnd   int
}

func sqliteWriter(inputs chan Row, wg *sync.WaitGroup) {
	defer wg.Done()
	db_url := "./tmp/sqlite.db"

	fmt.Println("Checking if sqlite file exists...")
	if _, err := os.Stat(db_url); err == nil {
		fmt.Println("File exists, deleting...")
		err = os.Remove(db_url)
		if err != nil {
			panic(fmt.Sprintf("failed to delete existing sqlite file: %v", err))
		}
		fmt.Println("File deleted.")
	} else {
		fmt.Println("File does not exist, nothing to delete.")
	}

	db, err := sql.Open("sqlite3", db_url)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	config := `
		PRAGMA journal_mode = OFF;
		PRAGMA synchronous = 0;
		PRAGMA cache_size = 1000000;
		PRAGMA locking_mode = EXCLUSIVE;
		PRAGMA temp_store = MEMORY;
	`

	_, err = db.Exec(config)
	if err != nil {
		panic(err)
	}

	create_table_query := `
	CREATE TABLE IF NOT EXISTS dev_table
	(
		id INTEGER PRIMARY KEY,
		bigint_column INTEGER,
		timestamp_start_column INTEGER,
		timestamp_end_column INTEGER
	)`
	_, err = db.Exec(create_table_query)
	if err != nil {
		panic(err)
	}

	// Prepare the INSERT statement once
	stmt, err := db.Prepare("INSERT INTO dev_table (id, bigint_column, timestamp_start_column, timestamp_end_column) VALUES (?, ?, ?, ?)")
	if err != nil {
		panic(err)
	}
	defer stmt.Close()

	const batchSize = 1000
	batch := make([]Row, 0, batchSize)

	// Function to process a batch
	processBatch := func(batch []Row) error {
		if len(batch) == 0 {
			return nil
		}

		// Begin transaction
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback() // Will be no-op if transaction is committed

		// Use prepared statement within transaction
		txStmt := tx.Stmt(stmt)

		for _, row := range batch {
			_, err := txStmt.Exec(row.Id, row.BigIntColumn, row.TimestampStart, row.TimestampEnd)
			if err != nil {
				return err
			}
		}

		// Commit transaction
		return tx.Commit()
	}

	// Process rows in batches
	for row := range inputs {
		batch = append(batch, row)

		// Process batch when it reaches the batch size
		if len(batch) >= batchSize {
			if err := processBatch(batch); err != nil {
				panic(err)
			}
			batch = batch[:0] // Reset batch slice but keep capacity
		}
	}

	// Process remaining rows in the final batch
	if err := processBatch(batch); err != nil {
		panic(err)
	}
}

func main() {
	db_url := "devuser:devpass@tcp(localhost:3306)/devdb"

	db, err := sql.Open("mysql", db_url)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	fmt.Println("Connected to MySQL")

	// Test connection
	if err := db.Ping(); err != nil {
		panic(err)
	}

	// Read all rows in batches of 1000
	batchSize := 50000
	offset := 0
	var totalRows uint64

	wg := sync.WaitGroup{}
	wg.Add(1)
	inputs := make(chan Row, batchSize)
	go sqliteWriter(inputs, &wg)

	for {
		query := "SELECT id, bigint_column, timestamp_start_column, timestamp_end_column FROM dev_table LIMIT ? OFFSET ?"
		rows, err := db.Query(query, batchSize, offset)
		if err != nil {
			panic(err)
		}

		rowsInBatch := 0
		for rows.Next() {
			var row Row
			err := rows.Scan(&row.Id, &row.BigIntColumn, &row.TimestampStart, &row.TimestampEnd)
			if err != nil {
				rows.Close()
				panic(err)
			}

			rowsInBatch++
			totalRows++

			inputs <- row
		}

		rows.Close()

		if err := rows.Err(); err != nil {
			panic(err)
		}

		// If we got fewer rows than the batch size, we've reached the end
		if rowsInBatch < batchSize {
			break
		}

		fmt.Printf("Total rows processed: %s\n", humanize.SI(float64(totalRows), ""))

		offset += batchSize
	}

	fmt.Printf("Finished processing all rows. Total rows processed: %d\n", totalRows)
	close(inputs)
	fmt.Println("Waiting for sqlite writer to finish...")
	wg.Wait()
	fmt.Println("Sqlite writer finished.")
}
