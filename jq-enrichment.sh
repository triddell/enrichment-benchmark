#!/bin/bash

# JQ enrichment script
gunzip -c test-data.ndjson.gz | jq -c --slurpfile routing aws-routing.json '
  . + (
    ($routing[0][.recipientAccountId] // {})
    | {
      lookup_target_pipeline: (.target_pipeline // "default_gis")
    }
  )
' | gzip > test-data.jq-output.ndjson.gz