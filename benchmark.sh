#!/bin/bash

# Performance benchmark script for comparing bento, jq, and Go enrichment

set -e

echo "========================================="
echo "CloudTrail Record Enrichment Benchmark"
echo "========================================="
echo ""

# Check if test data exists
if [ ! -f "test-data.ndjson.gz" ]; then
    echo "Error: test-data.ndjson.gz not found. Run generate_test_data.go first."
    exit 1
fi

# Check if routing file exists
if [ ! -f "aws-routing.json" ]; then
    echo "Error: aws-routing.json not found."
    exit 1
fi

# Show test data info
echo "Test Data:"
ls -lh test-data.ndjson.gz
RECORD_COUNT=$(gunzip -c test-data.ndjson.gz | wc -l | tr -d ' ')
echo "Records: $RECORD_COUNT"
echo ""

# Clean up previous outputs
rm -f test-data.bento-output.ndjson.gz test-data.jq-output.ndjson.gz test-data.go-output.ndjson.gz

# Benchmark function
run_benchmark() {
    local name=$1
    local command=$2

    echo "Testing: $name"
    echo "Command: $command"

    # Run 3 times and take the average
    local total=0
    for i in 1 2 3; do
        echo -n "  Run $i: "
        local start=$(date +%s%N)
        eval $command
        local end=$(date +%s%N)
        local duration=$((($end - $start) / 1000000))
        echo "${duration}ms"
        total=$(($total + $duration))
    done

    local avg=$(($total / 3))
    echo "  Average: ${avg}ms"
    echo ""
}

# Test 1: Bento
echo "========================================="
echo "1. Bento Enrichment"
echo "========================================="
if command -v bento &> /dev/null; then
    run_benchmark "Bento" "BENTO_FILE_PATH=test-data.ndjson.gz BENTO_OUTPUT_PATH=test-data.bento-output.ndjson bento -c bento-enrichment.yaml > /dev/null 2>&1 && gzip -f test-data.bento-output.ndjson"
    if [ -f "test-data.bento-output.ndjson.gz" ]; then
        echo "Output size:"
        ls -lh test-data.bento-output.ndjson.gz
        OUTPUT_RECORDS=$(gunzip -c test-data.bento-output.ndjson.gz | wc -l | tr -d ' ')
        echo "Output records: $OUTPUT_RECORDS"

        # Verify enrichment
        echo "Sample enriched record:"
        gunzip -c test-data.bento-output.ndjson.gz | head -1 | jq -c '{recipientAccountId, lookup_target_pipeline}'
    fi
else
    echo "Bento not found. Skipping..."
fi
echo ""

# Test 2: jq
echo "========================================="
echo "2. jq Enrichment"
echo "========================================="
if command -v jq &> /dev/null; then
    run_benchmark "jq" "./jq-enrichment.sh > /dev/null 2>&1"
    if [ -f "test-data.jq-output.ndjson.gz" ]; then
        echo "Output size:"
        ls -lh test-data.jq-output.ndjson.gz
        OUTPUT_RECORDS=$(gunzip -c test-data.jq-output.ndjson.gz | wc -l | tr -d ' ')
        echo "Output records: $OUTPUT_RECORDS"

        # Verify enrichment
        echo "Sample enriched record:"
        gunzip -c test-data.jq-output.ndjson.gz | head -1 | jq -c '{recipientAccountId, lookup_target_pipeline}'
    fi
else
    echo "jq not found. Skipping..."
fi
echo ""

# Test 3: Go
echo "========================================="
echo "3. Go Enrichment"
echo "========================================="
if command -v go &> /dev/null; then
    run_benchmark "Go" "go run go-enrichment.go > /dev/null 2>&1"
    if [ -f "test-data.go-output.ndjson.gz" ]; then
        echo "Output size:"
        ls -lh test-data.go-output.ndjson.gz
        OUTPUT_RECORDS=$(gunzip -c test-data.go-output.ndjson.gz | wc -l | tr -d ' ')
        echo "Output records: $OUTPUT_RECORDS"

        # Verify enrichment
        echo "Sample enriched record:"
        gunzip -c test-data.go-output.ndjson.gz | head -1 | jq -c '{recipientAccountId, lookup_target_pipeline}'
    fi
else
    echo "Go not found. Skipping..."
fi
echo ""

echo "========================================="
echo "Benchmark Complete"
echo "========================================="