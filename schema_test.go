package main

import "testing"

func TestSqliteType(t *testing.T) {
	tests := []struct {
		name       string
		mysqlType  string
		wantSQLite string
	}{
		// Integer types
		{"TINYINT", "TINYINT", "INTEGER"},
		{"TINYINT UNSIGNED", "TINYINT UNSIGNED", "INTEGER"},
		{"SMALLINT", "SMALLINT", "INTEGER"},
		{"SMALLINT(5)", "SMALLINT(5)", "INTEGER"},
		{"MEDIUMINT", "MEDIUMINT", "INTEGER"},
		{"INT", "INT", "INTEGER"},
		{"INT(11)", "INT(11)", "INTEGER"},
		{"INT UNSIGNED", "INT UNSIGNED", "INTEGER"},
		{"INTEGER", "INTEGER", "INTEGER"},
		{"BIGINT", "BIGINT", "INTEGER"},
		{"BIGINT(20) UNSIGNED", "BIGINT(20) UNSIGNED", "INTEGER"},
		{"BOOL", "BOOL", "INTEGER"},
		{"BOOLEAN", "BOOLEAN", "INTEGER"},

		// Floating point types
		{"FLOAT", "FLOAT", "REAL"},
		{"FLOAT(7,4)", "FLOAT(7,4)", "REAL"},
		{"DOUBLE", "DOUBLE", "REAL"},
		{"DOUBLE PRECISION", "DOUBLE PRECISION", "REAL"},
		{"DECIMAL", "DECIMAL", "REAL"},
		{"DECIMAL(10,2)", "DECIMAL(10,2)", "REAL"},
		{"NUMERIC", "NUMERIC", "REAL"},
		{"REAL", "REAL", "REAL"},

		// Binary types
		{"BLOB", "BLOB", "BLOB"},
		{"TINYBLOB", "TINYBLOB", "BLOB"},
		{"MEDIUMBLOB", "MEDIUMBLOB", "BLOB"},
		{"LONGBLOB", "LONGBLOB", "BLOB"},
		{"BINARY", "BINARY", "BLOB"},
		{"BINARY(16)", "BINARY(16)", "BLOB"},
		{"VARBINARY", "VARBINARY", "BLOB"},
		{"VARBINARY(255)", "VARBINARY(255)", "BLOB"},

		// Text types
		{"CHAR", "CHAR", "TEXT"},
		{"CHAR(10)", "CHAR(10)", "TEXT"},
		{"VARCHAR", "VARCHAR", "TEXT"},
		{"VARCHAR(255)", "VARCHAR(255)", "TEXT"},
		{"TEXT", "TEXT", "TEXT"},
		{"TINYTEXT", "TINYTEXT", "TEXT"},
		{"MEDIUMTEXT", "MEDIUMTEXT", "TEXT"},
		{"LONGTEXT", "LONGTEXT", "TEXT"},
		{"ENUM", "ENUM('a','b','c')", "TEXT"},
		{"SET", "SET('x','y','z')", "TEXT"},

		// Date/time types
		{"DATE", "DATE", "TEXT"},
		{"TIME", "TIME", "TEXT"},
		{"DATETIME", "DATETIME", "TEXT"},
		{"TIMESTAMP", "TIMESTAMP", "TEXT"},
		{"YEAR", "YEAR", "TEXT"},
		{"YEAR(4)", "YEAR(4)", "TEXT"},

		// JSON type
		{"JSON", "JSON", "TEXT"},

		// Case insensitivity
		{"lowercase int", "int", "INTEGER"},
		{"lowercase varchar", "varchar(100)", "TEXT"},
		{"lowercase blob", "blob", "BLOB"},
		{"UPPERCASE INT", "INT", "INTEGER"},
		{"MixedCase Int", "Int", "INTEGER"},
		{"MixedCase VarChar", "VarChar(50)", "TEXT"},

		// Type with modifiers
		{"INT UNSIGNED ZEROFILL", "INT UNSIGNED ZEROFILL", "INTEGER"},
		{"BIGINT AUTO_INCREMENT", "BIGINT AUTO_INCREMENT", "INTEGER"},
		{"VARCHAR CHARSET utf8mb4", "VARCHAR(255) CHARACTER SET utf8mb4", "TEXT"},

		// Unknown types (should default to TEXT)
		{"UNKNOWN", "UNKNOWN", "TEXT"},
		{"CUSTOM_TYPE", "CUSTOM_TYPE", "TEXT"},
		{"", "", "TEXT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sqliteType(tt.mysqlType)
			if got != tt.wantSQLite {
				t.Errorf("sqliteType(%q) = %q, want %q", tt.mysqlType, got, tt.wantSQLite)
			}
		})
	}
}
