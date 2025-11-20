# CloudTrail Record Enrichment Performance Benchmark

Performance comparison of three approaches for enriching CloudTrail records: **Bento**, **jq**, and **Go**.

## Overview

This benchmark tests adding a `lookup_target_pipeline` field to CloudTrail records by looking up `recipientAccountId` in a routing table.

**Test Dataset**: ~130,900 records (~5MB gzipped NDJSON)

## Results Summary

| Tool | Time | Records/sec | Relative Speed |
|------|------|-------------|----------------|
| **Go** | 2.5s | ~51,500 | **1.0x** (baseline) |
| **jq** | 5.3s | ~24,700 | **2.1x slower** |
| **Bento** | 38.7s | ~3,360 | **15.3x slower** |

### Bento Deep Dive

Bento's performance breakdown reveals interesting insights:

| Operation | Time | Notes |
|-----------|------|-------|
| Passthrough (no lookup) | 2.4s | Base I/O performance comparable to Go |
| With lookup enrichment | 38.7s | Lookup adds 36s overhead (277μs per record) |

**Key Finding**: Bento's file I/O and gzip handling is excellent (~2.4s), but the `file_rel()` lookup operation has significant per-record overhead compared to loading the lookup table once into native data structures.

## Prerequisites

```bash
# Bento
brew tap warpstreamlabs/bento
brew install bento

# jq
brew install jq

# Go
brew install go
```

## Quick Start

```bash
# Generate test data (~5MB gzipped)
go run generate_test_data.go

# Run the full benchmark
./benchmark.sh

# Run Bento-specific comparison (passthrough vs lookup)
./bento-comparison.sh
```

## Files

### Test Data Generation
- `generate_test_data.go` - Generates ~130,900 CloudTrail records
- `aws-routing.json` - Lookup table (1,812 account mappings)
- `cloudtrail-sample-record.json` - Example record structure

### Implementations

**Bento**
- `bento-enrichment.yaml` - Bento config with lookup enrichment
- `bento-passthrough.yaml` - Bento config for passthrough (no lookup)

**jq**
- `jq-enrichment.sh` - Shell script using jq with `--slurpfile`

**Go**
- `go-enrichment.go` - Go implementation with map-based lookup

### Benchmark Scripts
- `benchmark.sh` - Compares all three approaches (Bento, jq, Go)
- `bento-comparison.sh` - Bento-specific analysis (passthrough vs lookup)

## Implementation Details

### Bento
- Uses `file_rel()` with caching for routing table
- Decompress scanner for gzip input
- File-based I/O with environment variables
- **Lookup approach**: `file_rel()` + `parse_json()` + map access per record

### jq
- Loads routing table with `--slurpfile` (in-memory)
- Shell pipeline: `gunzip | jq | gzip`
- **Lookup approach**: One-time load into jq array, indexed access

### Go
- Loads routing table into `map[string]RoutingInfo` once
- Streaming JSON with `bufio.Scanner`
- Standard library gzip reader/writer
- **Lookup approach**: Native Go map loaded once, O(1) access per record

## Sample Output

All implementations correctly enrich records:

**Input:**
```json
{"recipientAccountId": "001885544633", ...}
```

**Output:**
```json
{"recipientAccountId": "001885544633", "lookup_target_pipeline": "default_pipeline", ...}
```

## Performance Analysis

### Why is Go Fastest?
1. Compiled binary (no interpretation overhead)
2. One-time lookup table load into native map
3. Efficient streaming JSON processing
4. Minimal allocation during processing

### Why is jq Good?
1. C-based implementation (fast)
2. One-time lookup table load with `--slurpfile`
3. Optimized JSON processing
4. Pipeline allows parallel decompression/compression

### Why is Bento Slower?
1. **Base I/O is excellent** (2.4s) - comparable to Go
2. **Lookup is the bottleneck** (adds 36s)
   - `file_rel()` may have per-record overhead despite caching
   - Bloblang mapping evaluation per record
   - Not as optimized as native map lookups

## Use Cases

**Choose Go when:**
- Raw performance is critical
- Batch processing large datasets
- You need minimal dependencies

**Choose jq when:**
- Shell scripting environment
- Quick prototyping
- Good balance of performance and simplicity

**Choose Bento when:**
- Building streaming data pipelines
- Need observability and monitoring
- Complex transformations beyond just lookups
- Continuous processing (not batch)

## License

This benchmark is provided as-is for educational and comparison purposes.
