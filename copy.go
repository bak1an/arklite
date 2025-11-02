package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/stephenafamo/bob/dialect/sqlite"
)

type RowData []any

type CopierOptions struct {
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

func NewCopier(mysqlDb *sql.DB, sqliteDb *sql.DB, schema *Schema, opts CopierOptions) *Copier {
	return &Copier{
		mysqlDb:  mysqlDb,
		sqliteDb: sqliteDb,
		schema:   schema,
		opts:     opts,
		wg:       sync.WaitGroup{},
	}
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

	idColumnIndex := c.schema.ColumnIndex(c.schema.IdColumn)
	if idColumnIndex == -1 {
		return fmt.Errorf("%s column not found", c.schema.IdColumn)
	}

	var batchStartAt time.Time
	var batchDuration time.Duration

	query := c.schema.MySQLSelectQuery(int64(c.opts.ReadBatchSize))
	stmt, err := c.mysqlDb.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for {
		batchStartAt = time.Now()
		rows, err := stmt.Query(maxSeenId)
		if err != nil {
			return err
		}
		rowsInBatch := 0
		for rows.Next() {
			row := make([]any, colsCount)
			for i := range colsCount {
				row[i] = reflect.New(c.schema.Columns[i].reflectType).Interface()
			}
			err := rows.Scan(row...)
			if err != nil {
				return err
			}
			rowsInBatch++

			switch row[idColumnIndex].(type) {
			case *int64:
				rowId := *row[idColumnIndex].(*int64)
				if rowId > maxSeenId {
					maxSeenId = rowId
				}
			case *int32:
				rowId := int64(*row[idColumnIndex].(*int32))
				if rowId > maxSeenId {
					maxSeenId = rowId
				}
			case *int16:
				rowId := int64(*row[idColumnIndex].(*int16))
				if rowId > maxSeenId {
					maxSeenId = rowId
				}
			default:
				return fmt.Errorf("unknown id column type: %T", row[idColumnIndex])
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
	for i, column := range c.schema.Columns {
		columns[i] = sqlite.Quote(column.name).String()
	}

	insertQuery := c.schema.SqliteInsertQuery()

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
