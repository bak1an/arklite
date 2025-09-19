#!/bin/bash

# Direct MySQL insertion script for 1 million test records
# This script runs as part of MySQL container initialization
# It will use the environment variables: MYSQL_DATABASE, MYSQL_USER, MYSQL_PASSWORD

set -e  # Exit on any error

echo "Starting bulk data insertion into ${MYSQL_DATABASE:-devdb}..."

# Configuration
TOTAL_ROWS=1000000
BATCH_SIZE=1000
MYSQL_HOST="localhost"
MYSQL_PORT="3306"
MYSQL_DB="${MYSQL_DATABASE:-devdb}"
MYSQL_USER="${MYSQL_USER:-devuser}"
MYSQL_PASS="${MYSQL_PASSWORD:-devpassword}"

# Date ranges for timestamps (2023-2024)
START_TIMESTAMP=1672531200  # 2023-01-01 00:00:00
END_TIMESTAMP=1735689599    # 2024-12-31 23:59:59
DATE_RANGE=$((END_TIMESTAMP - START_TIMESTAMP))

# Function to generate random number
random_bigint() {
    # Generate a random 18-digit bigint
    echo $((RANDOM * RANDOM * RANDOM + 1000000000000000))
}

# Function to generate random timestamp in range
random_timestamp() {
    local offset=$((RANDOM % DATE_RANGE))
    echo $((START_TIMESTAMP + offset))
}

echo "Starting data generation and insertion..."

# Generate and insert data in batches
for ((batch=0; batch<TOTAL_ROWS; batch+=BATCH_SIZE)); do
    # Create temporary SQL file for this batch
    TEMP_SQL=$(mktemp)

    echo "INSERT INTO \`dev_table\` (\`bigint_column\`, \`timestamp_start_column\`, \`timestamp_end_column\`) VALUES" > "$TEMP_SQL"

    # Generate batch data
    for ((i=0; i<BATCH_SIZE && (batch+i)<TOTAL_ROWS; i++)); do
        # Generate random values
        BIGINT_VALUE=$(random_bigint)
        TIMESTAMP_START=$(random_timestamp)

        # Generate duration between 1 hour and 30 days
        DURATION=$((RANDOM % 2592000 + 3600))  # 1 hour to 30 days
        TIMESTAMP_END=$((TIMESTAMP_START + DURATION))

        if [ $i -eq $((BATCH_SIZE-1)) ] || [ $((batch+i)) -eq $((TOTAL_ROWS-1)) ]; then
            # Last record in batch - no comma
            echo "($BIGINT_VALUE, $TIMESTAMP_START, $TIMESTAMP_END);" >> "$TEMP_SQL"
        else
            # Not last record - add comma
            echo "($BIGINT_VALUE, $TIMESTAMP_START, $TIMESTAMP_END)," >> "$TEMP_SQL"
        fi
    done

    # Execute the batch insert
    mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASS" "$MYSQL_DB" < "$TEMP_SQL"

    # Commit this batch
    mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASS" "$MYSQL_DB" -e "COMMIT;"

    # Clean up temp file
    rm "$TEMP_SQL"

    # Progress indicator
    if [ $((batch % 50000)) -eq 0 ]; then
        PROGRESS=$((batch + BATCH_SIZE))
        echo "Progress: $PROGRESS / $TOTAL_ROWS rows inserted ($(( PROGRESS * 100 / TOTAL_ROWS ))%)"
    fi
done

# Show final statistics
echo "Data insertion completed. Final statistics:"
mysql -h"$MYSQL_HOST" -P"$MYSQL_PORT" -u"$MYSQL_USER" -p"$MYSQL_PASS" "$MYSQL_DB" -e "
SELECT
    COUNT(*) as total_rows,
    MIN(id) as min_id,
    MAX(id) as max_id,
    FROM_UNIXTIME(MIN(timestamp_start_column)) as earliest_date,
    FROM_UNIXTIME(MAX(timestamp_end_column)) as latest_date
FROM dev_table;"

echo "Bulk data insertion completed successfully!"