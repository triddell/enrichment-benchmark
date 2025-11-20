package main

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
)

type RoutingInfo struct {
	AccountName    string `json:"account_name"`
	BusinessUnit   string `json:"business_unit"`
	Segment        string `json:"segment"`
	TargetPipeline string `json:"target_pipeline"`
}

func main() {
	// Load routing table
	routingFile, err := os.Open("aws-routing.json")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening routing file: %v\n", err)
		os.Exit(1)
	}
	defer routingFile.Close()

	var routing map[string]RoutingInfo
	if err := json.NewDecoder(routingFile).Decode(&routing); err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding routing file: %v\n", err)
		os.Exit(1)
	}

	// Open input file
	inputFile, err := os.Open("test-data.ndjson.gz")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer inputFile.Close()

	gzReader, err := gzip.NewReader(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating gzip reader: %v\n", err)
		os.Exit(1)
	}
	defer gzReader.Close()

	// Open output file
	outputFile, err := os.Create("test-data.go-output.ndjson.gz")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	gzWriter := gzip.NewWriter(outputFile)
	defer gzWriter.Close()

	scanner := bufio.NewScanner(gzReader)
	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	count := 0
	for scanner.Scan() {
		var record map[string]interface{}
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			fmt.Fprintf(os.Stderr, "Error decoding record: %v\n", err)
			continue
		}

		// Perform lookup
		recipientAccountId, ok := record["recipientAccountId"].(string)
		if ok && recipientAccountId != "" {
			if info, found := routing[recipientAccountId]; found {
				record["lookup_target_pipeline"] = info.TargetPipeline
			} else {
				record["lookup_target_pipeline"] = "default_gis"
			}
		} else {
			record["lookup_target_pipeline"] = "default_gis"
		}

		// Write enriched record
		enriched, err := json.Marshal(record)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding record: %v\n", err)
			continue
		}

		if _, err := gzWriter.Write(enriched); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing record: %v\n", err)
			os.Exit(1)
		}
		if _, err := gzWriter.Write([]byte("\n")); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing newline: %v\n", err)
			os.Exit(1)
		}

		count++
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning input: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Processed %d records\n", count)
}