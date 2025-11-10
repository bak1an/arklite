#!/usr/bin/env -S uv run --script

# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "mysql-connector-python",
# ]
# ///

import subprocess
import sys
import mysql.connector
import sqlite3
import os


def run_arklite() -> None:
    cmd = [
        "go",
        "run",
        ".",
        "-udevuser",
        "-pdevpass",
        "-ddevdb",
        "-tdevtable",
        "-o./tmp/out.db",
        "-f",  # Force overwrite
    ]
    result = subprocess.run(cmd, capture_output=True, text=True, cwd=os.getcwd())
    if result.returncode != 0:
        print("Error running arklite:")
        print(f"stdout: {result.stdout}")
        print(f"stderr: {result.stderr}")
        sys.exit(1)
    print("Created SQLite dump")


def get_mysql_connection():
    """Create MySQL connection."""
    return mysql.connector.connect(
        host="localhost",
        user="devuser",
        password="devpass",
        database="devdb",
    )


def get_sqlite_connection(db_path: str):
    """Create SQLite connection."""
    if not os.path.exists(db_path):
        print(f"Error: SQLite file not found at {db_path}")
        sys.exit(1)
    return sqlite3.connect(db_path)


def verify_row_count(mysql_cnx, sqlite_cnx) -> bool:
    """Verify that row counts match between MySQL and SQLite."""
    mysql_cursor = mysql_cnx.cursor(buffered=True)
    mysql_cursor.execute("SELECT COUNT(*) FROM devtable")
    mysql_count = mysql_cursor.fetchone()[0]
    mysql_cursor.close()

    sqlite_cursor = sqlite_cnx.cursor()
    sqlite_cursor.execute("SELECT COUNT(*) FROM devtable")
    sqlite_count = sqlite_cursor.fetchone()[0]
    sqlite_cursor.close()

    if mysql_count != sqlite_count:
        print(
            f"Error: Row counts do not match (MySQL: {mysql_count}, SQLite: {sqlite_count})"
        )
        return False

    print(f"Row counts match: {mysql_count}")
    return True


def verify_types(sqlite_cnx) -> bool:
    expected_types = tuple(
        ["integer", "integer", "text", "real", "text", "blob", "integer", "null"]
    )

    query = """SELECT DISTINCT
    typeof(id),
    typeof(bigint_column),
    typeof(datetime_column),
    typeof(float_column),
    typeof(string_column),
    typeof(blob_column),
    typeof(timestamp_column),
    typeof(nullable_column)
    FROM devtable
    """

    cursor = sqlite_cnx.cursor()
    cursor.execute(query)
    result = cursor.fetchall()
    cursor.close()

    if len(result) != 1:
        print(f"Error: Expected 1 row, got {len(result)}")
        print(f"Result: {result}")
        return False

    if result[0] != expected_types:
        print(f"Error: Type mismatch (expected {expected_types}, got {result[0]})")
        return False

    print(f"Types match: {expected_types}")
    return True


def verify_row_by_row(mysql_cnx, sqlite_cnx) -> bool:
    """Verify that all rows match between MySQL and SQLite."""
    mysql_cursor = mysql_cnx.cursor()
    sqlite_cursor = sqlite_cnx.cursor()

    # Get column names
    mysql_cursor.execute("SELECT * FROM devtable LIMIT 0")
    mysql_cursor.fetchall()
    mysql_columns = [desc[0] for desc in mysql_cursor.description]

    # Fetch rows lazily one by one, ordered by id
    mysql_query = """
    SELECT id, bigint_column, datetime_column, float_column,
           string_column, blob_column, timestamp_column, nullable_column
    FROM devtable
    ORDER BY id
    """

    sqlite_query = """
    SELECT id, bigint_column, datetime_column, float_column,
           string_column, blob_column, timestamp_column, nullable_column
    FROM devtable
    ORDER BY id
    """

    mysql_cursor.execute(mysql_query)
    sqlite_cursor.execute(sqlite_query)

    mismatches = 0
    row_count = 0

    while True:
        mysql_row = mysql_cursor.fetchone()
        sqlite_row = sqlite_cursor.fetchone()

        # Check if both cursors are exhausted
        if mysql_row is None and sqlite_row is None:
            break

        # Check if one cursor is exhausted before the other
        if mysql_row is None or sqlite_row is None:
            print(
                f"Error: Row count mismatch - one cursor exhausted before the other "
                f"(row {row_count + 1})"
            )
            mysql_cursor.close()
            sqlite_cursor.close()
            return False

        row_count += 1
        if len(mysql_row) != len(sqlite_row):
            print(
                f"Error: Column count mismatch (MySQL: {len(mysql_row)}, SQLite: {len(sqlite_row)})"
            )
            mismatches += 1
            continue

        row_id = mysql_row[0]
        row_mismatch = False
        for col_idx, (mysql_val, sqlite_val) in enumerate(zip(mysql_row, sqlite_row)):
            col_name = mysql_columns[col_idx]

            # Special handling for datetime
            if col_name == "datetime_column":
                # MySQL returns datetime objects, SQLite returns strings or bytes
                if isinstance(mysql_val, str):
                    mysql_str = mysql_val
                elif hasattr(mysql_val, "isoformat"):
                    # MySQL datetime object - convert to string format matching SQLite
                    mysql_str = mysql_val.strftime("%Y-%m-%d %H:%M:%S+00:00")
                else:
                    mysql_str = str(mysql_val)

                # SQLite may return bytes - decode to string if needed
                if sqlite_val is None:
                    sqlite_str = None
                elif isinstance(sqlite_val, bytes):
                    sqlite_str = sqlite_val.decode("utf-8")
                else:
                    sqlite_str = str(sqlite_val)

                if mysql_str != sqlite_str:
                    row_mismatch = True
                    print(
                        f"Error: Row {row_id}, column '{col_name}': "
                        f"MySQL={mysql_str}, SQLite={sqlite_str}"
                    )

            # Special handling for float columns
            elif col_name == "float_column":
                mysql_float = float(mysql_val) if mysql_val is not None else None
                sqlite_float = float(sqlite_val) if sqlite_val is not None else None
                if mysql_float is None and sqlite_float is None:
                    continue
                if mysql_float is None or sqlite_float is None:
                    row_mismatch = True
                    print(
                        f"Error: Row {row_id}, column '{col_name}': "
                        f"MySQL={mysql_float}, SQLite={sqlite_float}"
                    )
                elif (
                    abs(mysql_float - sqlite_float) > 0.0000000000001
                ):  # Allow small floating point differences
                    row_mismatch = True
                    print(
                        f"Error: Row {row_id}, column '{col_name}': "
                        f"MySQL={mysql_float}, SQLite={sqlite_float}"
                    )

            # Special handling for BLOB columns
            elif col_name == "blob_column":
                # MySQL connector returns bytes, SQLite returns bytes or memoryview
                mysql_bytes = bytes(mysql_val) if mysql_val is not None else None
                if isinstance(sqlite_val, memoryview):
                    sqlite_bytes = bytes(sqlite_val)
                elif sqlite_val is not None:
                    sqlite_bytes = bytes(sqlite_val)
                else:
                    sqlite_bytes = None

                if mysql_bytes != sqlite_bytes:
                    row_mismatch = True
                    print(
                        f"Error: Row {row_id}, column '{col_name}': "
                        f"MySQL length={len(mysql_bytes) if mysql_bytes else 0}, "
                        f"SQLite length={len(sqlite_bytes) if sqlite_bytes else 0}"
                    )

            # Direct comparison for other types (int, bigint, string, timestamp)
            else:
                # SQLite may return bytes for string columns - decode to string if needed
                mysql_comp = mysql_val
                if sqlite_val is None:
                    sqlite_comp = None
                elif isinstance(sqlite_val, bytes) and isinstance(mysql_val, str):
                    # SQLite returned bytes for a string column - decode it
                    sqlite_comp = sqlite_val.decode("utf-8")
                else:
                    sqlite_comp = sqlite_val

                if mysql_comp != sqlite_comp:
                    row_mismatch = True
                    print(
                        f"Error: Row {row_id}, column '{col_name}': "
                        f"MySQL={mysql_comp}, SQLite={sqlite_comp}"
                    )

        if row_mismatch:
            mismatches += 1

    mysql_cursor.close()
    sqlite_cursor.close()

    if mismatches > 0:
        print(f"Error: Found {mismatches} row(s) with mismatches")
        return False

    print(f"All {row_count} rows match")
    return True


def main() -> None:
    run_arklite()

    mysql_cnx = get_mysql_connection()
    sqlite_cnx = get_sqlite_connection("./tmp/out.db")

    try:
        if not verify_row_count(mysql_cnx, sqlite_cnx):
            sys.exit(1)

        if not verify_row_by_row(mysql_cnx, sqlite_cnx):
            sys.exit(1)

        if not verify_types(sqlite_cnx):
            sys.exit(1)

        print("All tests ok")

    finally:
        mysql_cnx.close()
        sqlite_cnx.close()


if __name__ == "__main__":
    main()
