package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

type ColumnInfo struct {
	name       string
	mysqlType  string
	sqliteType string
	scanType   reflect.Type
}

type Schema struct {
	Table   string
	Columns []*ColumnInfo
}

func ReadSchema(db *sql.DB, table string) (*Schema, error) {
	columnInfos, err := fetchColumnsInfo(db, table)
	if err != nil {
		return nil, err
	}
	schema := &Schema{
		Table:   table,
		Columns: columnInfos,
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

func (s *Schema) SQLiteCreateTableQuery() string {
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", s.Table)

	columns := make([]string, len(s.Columns))
	for i, columnInfo := range s.Columns {
		columns[i] = fmt.Sprintf("  %s %s", columnInfo.name, columnInfo.sqliteType)
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
	res, err := db.Query(fmt.Sprintf("SELECT * FROM %s WHERE 1=0", table))
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
			scanType:   column.ScanType(),
		}
	}
	return columnInfos, nil
}
