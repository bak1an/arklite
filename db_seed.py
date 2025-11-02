#!/usr/bin/env -S uv run --script

# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "mysql-connector-python",
# ]
# ///

import argparse
import random
import mysql.connector
import datetime
import string

# SQL query constants
CREATE_TABLE_QUERY = """
CREATE TABLE IF NOT EXISTS {table} (
    id INT AUTO_INCREMENT,
    bigint_column BIGINT,
    datetime_column DATETIME,
    float_column DOUBLE,
    string_column VARCHAR(255),
    blob_column BLOB,
    timestamp_column INT,
    PRIMARY KEY (id, datetime_column)
)
PARTITION BY RANGE (YEAR(datetime_column)) (
    PARTITION p_old VALUES LESS THAN (2000),
    PARTITION p_modern VALUES LESS THAN (2100),
    PARTITION p_future VALUES LESS THAN MAXVALUE
)
"""

DROP_TABLE_QUERY = """
DROP TABLE IF EXISTS {table}
"""

INSERT_DATA_QUERY = """
INSERT INTO {table} (bigint_column, datetime_column, float_column, string_column, blob_column, timestamp_column)
VALUES (%s, %s, %s, %s, %s, %s)
"""

COUNT_ROWS_QUERY = """
SELECT COUNT(*) FROM {table}
"""


def humanize_number(number: int) -> str:
    suffixes = ["", "k", "m"]
    for suffix in suffixes:
        if number < 1000:
            return f"{number}{suffix}"
        number /= 1000
        number = int(number) if int(number) == number else number
    return f"{number}b"


def main() -> None:
    # Parse command line arguments
    parser = argparse.ArgumentParser(description="Seed database with random data")
    parser.add_argument(
        "--count",
        "-c",
        type=int,
        default=100000,
        help="Number of random rows to insert (default: 100000)",
    )
    parser.add_argument(
        "--drop",
        "-d",
        action="store_true",
        help="Drop the table before inserting data",
    )
    parser.add_argument(
        "--table",
        "-t",
        default="devtable",
        help="Table to insert data into (default: devtable)",
    )
    args = parser.parse_args()
    cnx = mysql.connector.connect(
        host="localhost",
        user="devuser",
        password="devpass",
        database="devdb",
    )

    cursor = cnx.cursor()

    if args.drop:
        print("Dropping table...")
        cursor.execute(DROP_TABLE_QUERY.format(table=args.table))

    # Create table
    cursor.execute(CREATE_TABLE_QUERY.format(table=args.table))

    # Insert random data in batches
    batch_size = 10000
    print(
        f"Inserting {humanize_number(args.count)} random rows in batches of {humanize_number(batch_size)}..."
    )

    for batch_start in range(0, args.count, batch_size):
        batch_end = min(batch_start + batch_size, args.count)
        batch_data = []

        # Generate batch data
        for i in range(batch_start, batch_end):
            # Generate random bigint
            bigint_value = random.randint(1, 1000000)

            # Generate random datetime across all three partitions
            partition_choice = random.choice(["old", "modern", "future"])
            if partition_choice == "old":
                year = random.randint(1900, 1999)
            elif partition_choice == "modern":
                year = random.randint(2000, 2099)
            else:  # future
                year = random.randint(2100, 2200)

            month = random.randint(1, 12)
            day = random.randint(1, 28)  # Use 28 to avoid invalid dates
            hour = random.randint(0, 23)
            minute = random.randint(0, 59)
            second = random.randint(0, 59)

            datetime_value = datetime.datetime(year, month, day, hour, minute, second)

            # Generate random float
            float_value = round(random.uniform(-1000.0, 1000.0), 2)

            # Generate random string
            string_length = random.randint(10, 255)
            string_value = "".join(
                random.choices(
                    string.ascii_letters + string.digits + " ", k=string_length
                )
            )

            # Generate random blob data
            blob_size = random.randint(10, 1000)
            blob_value = bytes([random.randint(0, 255) for _ in range(blob_size)])

            # Generate random timestamp (Unix timestamp)
            timestamp_value = random.randint(1000000000, 2000000000)

            batch_data.append(
                (
                    bigint_value,
                    datetime_value,
                    float_value,
                    string_value,
                    blob_value,
                    timestamp_value,
                )
            )

        # Insert entire batch at once
        cursor.executemany(INSERT_DATA_QUERY.format(table=args.table), batch_data)

        # Print progress every batch
        print(f"Inserted {humanize_number(batch_end)} rows...")

    cnx.commit()

    # Count and print total rows
    cursor.execute(COUNT_ROWS_QUERY.format(table=args.table))
    count = cursor.fetchone()[0]
    print(f"Total rows in {args.table}: {humanize_number(count)}")

    cursor.close()
    cnx.close()


if __name__ == "__main__":
    main()
