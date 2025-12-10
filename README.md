# arklite

Archive MySQL tables (or parts of them) into SQLite.

Not a backup tool.

## Why?

When:

1. your MySQL table becomes too big
2. you do not really need _all_ the data for daily operations
3. but also can not delete extra data forever just yet

You might want to archive not needed data to some cold storage and cleanup original table in MySQL.

This tool can help with archival part. Just like percona's [pt-archiver](https://docs.percona.com/percona-toolkit/pt-archiver.html) but smaller and with SQLite for output files.

Deletion of archived rows from MySQL is up to you (at least in this version) as this can be tricky when under load and there are many methods of doing it; see pt-archiver's help page for approximate list of things that can go wrong, I can also remember a few stories.

SQLite was chosen as output format as it is stable, portable and can be easily queried from any language or even bash script if needed.

And it can go reasonably fast (in my cases 50k-300k rows/s depending on table structure).

## Installation

```bash
go install github.com/bak1an/arklite@latest
```

Or download a binary from releases page.

## Usage

```bash
arklite -u <user> -d <database> -t <table> -o <output.sqlite>
```

It will read table schema from MySQL and will create _similar_ schema for SQLite.

SQLite table will have the same columns, similar types (where possible) and an index on id column.
All the rest of indexes and constraints seen in MySQL table will be ignored.

Give it `--preview` flag to dry run and see queries it is going to execute without doing anything.

Reading from MySQL will be done in batches with queries like:

```sql
SELECT ... FROM ...
WHERE some_id_column > previously_seen_max_id
ORDER BY some_id_column ASC
LIMIT ...
```

By default id column will be `id` but you can customize that with options.

## Required Flags

- `-u, --user` - MySQL user
- `-d, --database` - MySQL database name
- `-t, --table` - MySQL table name
- `-o, --output` - SQLite output file path

## Optional Flags

### Connection

- `-H, --host` - MySQL host (default: localhost)
- `-P, --port` - MySQL port (default: 3306)
- `-p, --password` - MySQL password
- `--ask-password` - Prompt for password interactively

### Data Filtering

- `--where` - WHERE clause filter (can be used multiple times)
- `--partition` - MySQL partition to copy
- `--only-columns` - Copy only specified columns (comma-separated)
- `--exclude-columns` - Exclude specified columns (comma-separated)
- `--limit` - Limit total number of rows to copy (0 = no limit)
- `--id-column` - ID column for pagination and ordering (default: "id")

### Performance

- `--read-batch` - Read batch size (default: 100000)
- `--write-batch` - Write batch size (default: 10000)

### Other Options

- `-f, --force` - Force overwrite existing SQLite file
- `--preview` - Preview SQL queries without copying data
- `--no-progress` - Disable progress bar
- `--verbose` - Enable verbose output
- `-v, --version` - Print version information

## Example

```bash
# Basic usage
arklite -u root -d mydb -t users -o users.sqlite

# With filtering and limit
arklite -u root -d mydb -t users -o users.sqlite \
  --where "created_at > '2024-01-01'" \
  --limit 10000

# Copy only specific columns
arklite -u root -d mydb -t users -o users.sqlite \
  --only-columns "id,username,email"

# Preview queries before copying
arklite -u root -d mydb -t users -o users.sqlite --preview
```
