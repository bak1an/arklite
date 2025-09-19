package main

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

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
	totalRows := 0

	for {
		fmt.Printf("Reading batch starting at offset %d...\n", offset)

		query := "SELECT id, bigint_column, timestamp_start_column, timestamp_end_column FROM dev_table LIMIT ? OFFSET ?"
		rows, err := db.Query(query, batchSize, offset)
		if err != nil {
			panic(err)
		}

		rowsInBatch := 0
		for rows.Next() {
			var id int64
			var bigintColumn uint64
			var timestampStart int
			var timestampEnd int

			err := rows.Scan(&id, &bigintColumn, &timestampStart, &timestampEnd)
			if err != nil {
				rows.Close()
				panic(err)
			}

			rowsInBatch++
			totalRows++

			// You can process each row here
			// fmt.Printf("Row %d: ID=%d, BigInt=%d, StartTime=%d, EndTime=%d\n",
			// totalRows, id, bigintColumn, timestampStart, timestampEnd)
		}

		rows.Close()

		if err := rows.Err(); err != nil {
			panic(err)
		}

		fmt.Printf("Processed %d rows in this batch\n", rowsInBatch)

		// If we got fewer rows than the batch size, we've reached the end
		if rowsInBatch < batchSize {
			break
		}

		offset += batchSize
	}

	fmt.Printf("Finished processing all rows. Total rows processed: %d\n", totalRows)
}
