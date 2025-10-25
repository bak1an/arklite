package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
)

type RowData []any

type CopierOptions struct {
	Table          string
	Partition      string
	WriteBatchSize int
	ReadBatchSize  int
}

type Copier struct {
	mysqlDb  *sql.DB
	sqliteDb *sql.DB
	opts     CopierOptions
	schema   *Schema
	wg       sync.WaitGroup
}

func NewCopier(mysqlDb *sql.DB, sqliteDb *sql.DB, opts CopierOptions) (*Copier, error) {
	schema, err := ReadSchema(mysqlDb, opts.Table)
	if err != nil {
		return nil, err
	}
	cp := &Copier{
		mysqlDb:  mysqlDb,
		sqliteDb: sqliteDb,
		opts:     opts,
		schema:   schema,
		wg:       sync.WaitGroup{},
	}
	return cp, nil
}

func (c *Copier) CreateTable() error {
	slog.Info("Creating SQLite table", "table", c.schema.Table)
	query := c.schema.SQLiteCreateTableQuery()
	slog.Debug("SQLite create table query")
	slog.Debug(query)
	_, err := c.sqliteDb.Exec(query)
	if err != nil {
		return err
	}
	slog.Info("SQLite table created successfully", "table", c.schema.Table)
	return nil
}

func (c *Copier) Wait() {
	slog.Info("Wrapping up...")
	c.wg.Wait()
}

func (c *Copier) Copy() error {
	slog.Info("Copying data from MySQL to SQLite", "table", c.schema.Table)

	rowsChan := make(chan RowData, c.opts.WriteBatchSize*10) // big channel, let mysql read fast if it can
	defer close(rowsChan)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		err := c.sqliteWriter(rowsChan)
		if err != nil {
			slog.Error("Error writing to SQLite", "error", err)
			panic(err) // TODO: handle it a bit better
		}

	}()

	var maxSeenId int64 = 0
	colsCount := len(c.schema.Columns)
	colsNames := make([]string, colsCount)
	for i := range colsCount {
		colsNames[i] = c.schema.Columns[i].name
	}
	allColumns := strings.Join(colsNames, ", ")
	idColumnIndex := c.schema.ColumnIndex("id")
	if idColumnIndex == -1 {
		return fmt.Errorf("id column not found")
	}

	var batchStartAt time.Time
	var batchDuration time.Duration

	for {
		batchStartAt = time.Now()
		query := fmt.Sprintf(
			"SELECT %s FROM %s WHERE id > ? ORDER BY id ASC LIMIT ?",
			allColumns,
			c.schema.Table,
		)
		rows, err := c.mysqlDb.Query(query, maxSeenId, c.opts.ReadBatchSize)
		if err != nil {
			return err
		}
		rowsInBatch := 0
		for rows.Next() {
			row := make([]any, colsCount)
			ptrs := make([]any, colsCount)
			for i := range colsCount {
				ptrs[i] = &row[i]
			}
			err := rows.Scan(ptrs...)
			if err != nil {
				return err
			}
			rowsInBatch++

			rowId := row[idColumnIndex].(int64)
			if rowId > maxSeenId {
				maxSeenId = rowId
			}

			rowsChan <- row
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}

		count, suffix := humanize.ComputeSI(float64(rowsInBatch))
		rowsInBatchHumanized := fmt.Sprintf("%d%s", int(count), suffix)
		batchDuration = time.Since(batchStartAt)
		slog.Info(
			"Batch read from MySQL",
			"batch_duration", batchDuration,
			"rows_in_batch", rowsInBatchHumanized,
		)

		if rowsInBatch < c.opts.ReadBatchSize {
			break
		}
	}
	return nil
}

func (c *Copier) sqliteWriter(inputs <-chan RowData) error {
	columns := make([]string, len(c.schema.Columns))
	placeholders := make([]string, len(c.schema.Columns))
	for i, column := range c.schema.Columns {
		columns[i] = column.name
		placeholders[i] = "?"
	}
	insertQuery := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		c.schema.Table,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)
	stmt, err := c.sqliteDb.Prepare(insertQuery)
	if err != nil {
		return err
	}
	defer stmt.Close()

	batch := make([]RowData, 0, c.opts.WriteBatchSize)

	processBatch := func(batch []RowData) error {
		batchStartAt := time.Now()
		if len(batch) == 0 {
			return nil
		}
		// Begin transaction
		tx, err := c.sqliteDb.Begin()
		if err != nil {
			return err
		}
		defer tx.Rollback() // Will be no-op if transaction is committed

		// Use prepared statement within transaction
		txStmt := tx.Stmt(stmt)

		for _, row := range batch {
			_, err := txStmt.Exec(row...)
			if err != nil {
				return err
			}
		}

		// Commit transaction
		err = tx.Commit()
		if err != nil {
			return err
		}

		batchDuration := time.Since(batchStartAt)
		count, suffix := humanize.ComputeSI(float64(len(batch)))
		batchSizeHumanized := fmt.Sprintf("%d%s", int(count), suffix)
		slog.Debug(
			"Batch written to SQLite",
			"batch_duration", batchDuration,
			"batch_size", batchSizeHumanized,
		)

		return nil
	}
	for row := range inputs {
		batch = append(batch, row)

		// Process batch when it reaches the batch size
		if len(batch) >= c.opts.WriteBatchSize {
			if err := processBatch(batch); err != nil {
				return err
			}
			batch = batch[:0] // Reset batch slice but keep capacity
		}
	}

	// Process remaining rows in the final batch
	if err := processBatch(batch); err != nil {
		return err
	}

	slog.Info("SQLite writer finished")

	return nil
}
