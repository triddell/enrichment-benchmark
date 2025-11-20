#!/bin/bash

# Bento performance comparison: passthrough vs lookup enrichment

set -e

echo "========================================="
echo "Bento Performance: Passthrough vs Lookup"
echo "========================================="
echo ""

# Check if test data exists
if [ ! -f "test-data.ndjson.gz" ]; then
    echo "Error: test-data.ndjson.gz not found."
    exit 1
fi

# Show test data info
echo "Test Data:"
ls -lh test-data.ndjson.gz
RECORD_COUNT=$(gunzip -c test-data.ndjson.gz | wc -l | tr -d ' ')
echo "Records: $RECORD_COUNT"
echo ""

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

    echo $avg
}

# Clean up previous outputs
rm -f test-data.bento-passthrough.ndjson test-data.bento-passthrough.ndjson.gz
rm -f test-data.bento-lookup.ndjson test-data.bento-lookup.ndjson.gz

# Test 1: Bento Passthrough (no lookup)
echo "========================================="
echo "1. Bento Passthrough (No Lookup)"
echo "========================================="
PASSTHROUGH_TIME=$(run_benchmark "Bento Passthrough" "BENTO_FILE_PATH=test-data.ndjson.gz BENTO_OUTPUT_PATH=test-data.bento-passthrough.ndjson bento -c bento-passthrough.yaml > /dev/null 2>&1 && gzip -f test-data.bento-passthrough.ndjson" | tail -1)

if [ -f "test-data.bento-passthrough.ndjson.gz" ]; then
    echo "Output size:"
    ls -lh test-data.bento-passthrough.ndjson.gz
    OUTPUT_RECORDS=$(gunzip -c test-data.bento-passthrough.ndjson.gz | wc -l | tr -d ' ')
    echo "Output records: $OUTPUT_RECORDS"
    echo ""
fi

# Test 2: Bento with Lookup
echo "========================================="
echo "2. Bento with Lookup Enrichment"
echo "========================================="
LOOKUP_TIME=$(run_benchmark "Bento with Lookup" "BENTO_FILE_PATH=test-data.ndjson.gz BENTO_OUTPUT_PATH=test-data.bento-lookup.ndjson bento -c bento-enrichment.yaml > /dev/null 2>&1 && gzip -f test-data.bento-lookup.ndjson" | tail -1)

if [ -f "test-data.bento-lookup.ndjson.gz" ]; then
    echo "Output size:"
    ls -lh test-data.bento-lookup.ndjson.gz
    OUTPUT_RECORDS=$(gunzip -c test-data.bento-lookup.ndjson.gz | wc -l | tr -d ' ')
    echo "Output records: $OUTPUT_RECORDS"

    # Verify enrichment
    echo "Sample enriched record:"
    gunzip -c test-data.bento-lookup.ndjson.gz | head -1 | jq -c '{recipientAccountId, lookup_target_pipeline}'
    echo ""
fi

echo "========================================="
echo "Summary"
echo "========================================="
echo "Bento Passthrough (no lookup):  ${PASSTHROUGH_TIME}ms avg"
echo "Bento with Lookup:               ${LOOKUP_TIME}ms avg"
echo ""

if [ $LOOKUP_TIME -gt 0 ] && [ $PASSTHROUGH_TIME -gt 0 ]; then
    OVERHEAD=$((LOOKUP_TIME - PASSTHROUGH_TIME))
    PERCENT=$((OVERHEAD * 100 / PASSTHROUGH_TIME))
    echo "Lookup overhead:                 ${OVERHEAD}ms (${PERCENT}% increase)"
    echo ""
    echo "Analysis:"
    echo "  - Base processing time:        ${PASSTHROUGH_TIME}ms"
    echo "  - Lookup operation cost:       ${OVERHEAD}ms"
    echo "  - Cost per record:             $((OVERHEAD * 1000 / RECORD_COUNT))μs"
fi

echo ""
echo "========================================="
echo "Comparison Complete"
echo "========================================="