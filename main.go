package main

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/bak1an/arklite/config"
	"github.com/spf13/pflag"
	"golang.org/x/term"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/mattn/go-sqlite3"
)

const sqliteConfigQuery = `
		PRAGMA journal_mode = OFF;
		PRAGMA synchronous = 0;
		PRAGMA cache_size = 1000000;
		PRAGMA locking_mode = EXCLUSIVE;
		PRAGMA temp_store = MEMORY;
	`

func main() {
	mysqlHost := pflag.StringP("host", "H", "localhost", "MySQL host")
	mysqlPort := pflag.IntP("port", "P", 3306, "MySQL port")
	mysqlUser := pflag.StringP("user", "u", "", "(required) MySQL user")
	mysqlPassword := pflag.StringP("password", "p", "", "MySQL password")
	askPassword := pflag.Bool("ask-password", false, "Ask for MySQL password")
	mysqlDatabase := pflag.StringP("database", "d", "", "(required) MySQL database")
	mysqlTable := pflag.StringP("table", "t", "", "(required) MySQL table")
	sqliteFile := pflag.StringP("output", "o", "", "(required) SQLite file to write to")
	forceOverwrite := pflag.BoolP("force", "f", false, "Force overwrite existing SQLite file")
	partition := pflag.String("partition", "", "MySQL partition to copy")
	writeBatchSize := pflag.Int("write-batch", 10000, "Write batch size")
	readBatchSize := pflag.Int("read-batch", 100000, "Read batch size")
	preview := pflag.Bool("preview", false, "Preivew the SQL queries. Does not perform actual data copy.")
	verbose := pflag.Bool("verbose", false, "Verbose output")
	version := pflag.BoolP("version", "v", false, "Print version info")

	pflag.CommandLine.SortFlags = false

	pflag.Parse()

	if *askPassword {
		if *mysqlPassword == "" {
			fmt.Print("Enter password: ")
			passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			if err != nil {
				fmt.Println("Error reading password:", err)
				os.Exit(1)
			}
			fmt.Println() // Print newline after password input
			*mysqlPassword = string(passwordBytes)
		}
	}

	if *version {
		buildInfo := config.GetBuildInfo()
		fmt.Printf("arklite %s-%s\n", buildInfo.GitBranch, buildInfo.GitRev)
		fmt.Printf("build on %s with go %s\n", buildInfo.BuildTime, buildInfo.GoVersion)
		os.Exit(0)
	}

	if *verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
	} else {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	}

	if *mysqlDatabase == "" || *mysqlTable == "" || *sqliteFile == "" || *mysqlUser == "" {
		pflag.Usage()
		fmt.Println("Required flags are missing:")
		if *mysqlDatabase == "" {
			fmt.Println("  --database, -d <database>")
		}
		if *mysqlTable == "" {
			fmt.Println("  --table, -t <table>")
		}
		if *sqliteFile == "" {
			fmt.Println("  --output, -o <file>")
		}
		if *mysqlUser == "" {
			fmt.Println("  --user, -u <user>")
		}
		os.Exit(1)
	}

	userPart := *mysqlUser
	if *mysqlPassword != "" {
		userPart += ":" + *mysqlPassword
	}

	mysqlUrl := fmt.Sprintf("%s@tcp(%s:%d)/%s", userPart, *mysqlHost, *mysqlPort, *mysqlDatabase)

	mysqlDb, err := sql.Open("mysql", mysqlUrl)
	if err != nil {
		slog.Error("Error connecting to MySQL", "error", err)
		os.Exit(1)
	}
	defer mysqlDb.Close()

	// Test connection
	if err := mysqlDb.Ping(); err != nil {
		slog.Error("Error pinging MySQL", "error", err)
		os.Exit(1)
	}

	if *preview {
		fmt.Println("Queries to be executed:")
		schema, err := ReadSchema(mysqlDb, *mysqlTable, *partition)
		if err != nil {
			slog.Error("Error reading schema", "error", err)
			os.Exit(1)
		}
		createTableQuery := schema.SQLiteCreateTableQuery()
		selectQuery := schema.MySQLSelectQuery(int64(*readBatchSize))
		fmt.Printf(
			"\nWill create sqlite table in %s with:\n%s\n\n",
			*sqliteFile, createTableQuery,
		)
		fmt.Printf("Will select data from MySQL with:\n%s\n\n", selectQuery)

		insertQuery := schema.SqliteInsertQuery()
		fmt.Printf("Will insert data into SQLite with:\n%s\n\n", insertQuery)

		fmt.Printf(
			"Reads in batches of %d rows from MySQL and writes to SQLite in batches of %d rows.\n",
			*readBatchSize, *writeBatchSize,
		)
		os.Exit(0)
	}

	if _, err := os.Stat(*sqliteFile); err == nil {
		if !*forceOverwrite {
			slog.Error("SQLite file already exists, use --force to overwrite")
			os.Exit(1)
		}
		err = os.Remove(*sqliteFile)
		if err != nil {
			slog.Error("Error removing existing SQLite file", "error", err)
			os.Exit(1)
		}
	}

	sqliteDb, err := sql.Open("sqlite3", *sqliteFile)
	if err != nil {
		slog.Error("Error opening SQLite file", "error", err)
		os.Exit(1)
	}
	defer sqliteDb.Close()

	_, err = sqliteDb.Exec(sqliteConfigQuery)
	if err != nil {
		slog.Error("Error configuring SQLite:", "error", err)
		os.Exit(1)
	}

	copierOpts := CopierOptions{
		Table:          *mysqlTable,
		WriteBatchSize: *writeBatchSize,
		ReadBatchSize:  *readBatchSize,
		Partition:      *partition,
	}
	copier, err := NewCopier(mysqlDb, sqliteDb, copierOpts)
	if err != nil {
		slog.Error("Error creating copier", "error", err)
		os.Exit(1)
	}

	err = copier.CreateTable()
	if err != nil {
		slog.Error("Error creating table", "error", err)
		os.Exit(1)
	}
	err = copier.Copy()
	if err != nil {
		slog.Error("Error copying data", "error", err)
		os.Exit(1)
	}
	copier.Wait()
}
