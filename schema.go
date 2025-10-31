package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/dialect/sqlite"
)

type ColumnInfo struct {
	name       string
	mysqlType  string
	sqliteType string
}

type Schema struct {
	Table       string
	Partition   string
	IdColumn    string
	Columns     []*ColumnInfo
	columnNames []string
	allColumns  string
}

func ReadSchema(db *sql.DB, table string, partition string) (*Schema, error) {
	columnInfos, err := fetchColumnsInfo(db, table)
	if err != nil {
		return nil, err
	}
	columnNames := make([]string, len(columnInfos))
	for i, column := range columnInfos {
		columnNames[i] = column.name
	}
	schema := &Schema{
		Table:       table,
		Columns:     columnInfos,
		Partition:   partition,
		IdColumn:    "id",
		columnNames: columnNames,
		allColumns:  strings.Join(columnNames, ", "),
	}
	return schema, nil
}

func (s *Schema) ColumnIndex(name string) int {
	for i, column := range s.Columns {
		if column.name == name {
			return i
		}
	}
	return -1
}

func (s *Schema) MySQLSelectQuery(limit int64) string {
	from := sm.From(mysql.Quote(s.Table))
	if s.Partition != "" {
		from = from.Partition(s.Partition)
	}

	cols := make([]any, len(s.columnNames))
	for i, column := range s.columnNames {
		cols[i] = mysql.Quote(column)
	}

	q := mysql.Select(
		from,
		sm.Columns(cols...),
		sm.Where(mysql.Quote(s.IdColumn).GT(mysql.Placeholder(1))),
		sm.OrderBy(mysql.Quote(s.IdColumn)).Asc(),
		sm.Limit(limit),
	)
	sql, _, err := q.Build(context.Background())
	if err != nil {
		return ""
	}

	return sql
}

func (s *Schema) SQLiteCreateTableQuery() string {
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", sqlite.Quote(s.Table))

	columns := make([]string, len(s.Columns))
	for i, columnInfo := range s.Columns {
		columns[i] = fmt.Sprintf("  %s %s", sqlite.Quote(columnInfo.name), columnInfo.sqliteType)
		if columnInfo.name == "id" {
			columns[i] += " PRIMARY KEY AUTOINCREMENT"
		}
	}
	query += strings.Join(columns, ",\n")
	query += "\n)"
	return query
}

func sqliteType(mysqlType string) string {
	// Map MySQL types to SQLite types
	// SQLite has a simple type system: TEXT, INTEGER, REAL, BLOB
	// Use substring matching to handle type modifiers like UNSIGNED, ZEROFILL, etc.
	typeUpper := strings.ToUpper(mysqlType)

	// Integer types (check more specific types first)
	if strings.Contains(typeUpper, "TINYINT") || strings.Contains(typeUpper, "SMALLINT") ||
		strings.Contains(typeUpper, "MEDIUMINT") || strings.Contains(typeUpper, "BIGINT") ||
		strings.Contains(typeUpper, "INT") || strings.Contains(typeUpper, "INTEGER") ||
		strings.Contains(typeUpper, "BOOL") {
		return "INTEGER"
	}

	// Floating point types
	if strings.Contains(typeUpper, "FLOAT") || strings.Contains(typeUpper, "DOUBLE") ||
		strings.Contains(typeUpper, "DECIMAL") || strings.Contains(typeUpper, "NUMERIC") ||
		strings.Contains(typeUpper, "REAL") {
		return "REAL"
	}

	// Binary types (check before TEXT to avoid BLOB matching as TEXT)
	if strings.Contains(typeUpper, "BLOB") || strings.Contains(typeUpper, "BINARY") {
		return "BLOB"
	}

	// String/text types
	if strings.Contains(typeUpper, "CHAR") || strings.Contains(typeUpper, "TEXT") ||
		strings.Contains(typeUpper, "ENUM") || strings.Contains(typeUpper, "SET") {
		return "TEXT"
	}

	// Date/time types (SQLite stores as TEXT or INTEGER)
	if strings.Contains(typeUpper, "DATE") || strings.Contains(typeUpper, "TIME") ||
		strings.Contains(typeUpper, "TIMESTAMP") || strings.Contains(typeUpper, "YEAR") {
		return "TEXT"
	}

	// JSON type (MySQL 5.7+)
	if strings.Contains(typeUpper, "JSON") {
		return "TEXT"
	}

	// Default to TEXT for unknown types
	return "TEXT"
}

func fetchColumnsInfo(db *sql.DB, table string) ([]*ColumnInfo, error) {
	q := mysql.Select(
		sm.From(mysql.Quote(table)),
		sm.Where(mysql.Raw("1 = 0")),
		sm.Limit(1),
	)

	sql, _, err := q.Build(context.Background())
	if err != nil {
		return nil, err
	}

	res, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer res.Close()

	columnTypes, err := res.ColumnTypes()
	if err != nil {
		return nil, err
	}

	columnInfos := make([]*ColumnInfo, len(columnTypes))
	for i, column := range columnTypes {
		columnType := column.DatabaseTypeName()
		columnInfos[i] = &ColumnInfo{
			name:       column.Name(),
			mysqlType:  columnType,
			sqliteType: sqliteType(columnType),
		}
	}
	return columnInfos, nil
}
