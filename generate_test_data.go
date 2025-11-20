package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
)

type CloudTrailRecord struct {
	AdditionalEventData map[string]interface{} `json:"additionalEventData"`
	AwsRegion           string                 `json:"awsRegion"`
	EventCategory       string                 `json:"eventCategory"`
	EventID             string                 `json:"eventID"`
	EventName           string                 `json:"eventName"`
	EventSource         string                 `json:"eventSource"`
	EventTime           string                 `json:"eventTime"`
	EventType           string                 `json:"eventType"`
	EventVersion        string                 `json:"eventVersion"`
	ManagementEvent     bool                   `json:"managementEvent"`
	ReadOnly            bool                   `json:"readOnly"`
	RecipientAccountId  string                 `json:"recipientAccountId"`
	RequestID           string                 `json:"requestID"`
	RequestParameters   map[string]interface{} `json:"requestParameters"`
	Resources           []map[string]interface{} `json:"resources"`
	ResponseElements    map[string]interface{} `json:"responseElements"`
	SharedEventID       string                 `json:"sharedEventID"`
	SourceIPAddress     string                 `json:"sourceIPAddress"`
	TlsDetails          map[string]interface{} `json:"tlsDetails"`
	UserAgent           string                 `json:"userAgent"`
	UserIdentity        map[string]interface{} `json:"userIdentity"`
}

var accountIDs []string

func loadAccountIDs() error {
	f, err := os.Open("aws-routing.json")
	if err != nil {
		return err
	}
	defer f.Close()

	var routing map[string]interface{}
	if err := json.NewDecoder(f).Decode(&routing); err != nil {
		return err
	}

	// Extract account IDs from routing table
	for accountID := range routing {
		accountIDs = append(accountIDs, accountID)
	}

	// Use some but not all accounts (to test default case)
	if len(accountIDs) > 20 {
		accountIDs = accountIDs[:20]
	}

	return nil
}

var eventNames = []string{
	"AssumeRole",
	"GetObject",
	"PutObject",
	"DeleteObject",
	"ListBucket",
	"CreateBucket",
	"DescribeInstances",
	"RunInstances",
	"TerminateInstances",
}

var regions = []string{
	"us-west-2",
	"us-east-1",
	"eu-west-1",
	"ap-southeast-1",
}

func generateRecord() CloudTrailRecord {
	return CloudTrailRecord{
		AdditionalEventData: map[string]interface{}{
			"ExtendedRequestId": fmt.Sprintf("EXT-%d", rand.Intn(1000000)),
			"RequestDetails": map[string]interface{}{
				"awsServingRegion": regions[rand.Intn(len(regions))],
				"endpointType":     "regional",
			},
		},
		AwsRegion:          regions[rand.Intn(len(regions))],
		EventCategory:      "Management",
		EventID:            fmt.Sprintf("event-%d", rand.Intn(1000000)),
		EventName:          eventNames[rand.Intn(len(eventNames))],
		EventSource:        "sts.amazonaws.com",
		EventTime:          "2025-09-25T16:50:47Z",
		EventType:          "AwsApiCall",
		EventVersion:       "1.08",
		ManagementEvent:    true,
		ReadOnly:           true,
		RecipientAccountId: accountIDs[rand.Intn(len(accountIDs))],
		RequestID:          fmt.Sprintf("req-%d", rand.Intn(1000000)),
		RequestParameters: map[string]interface{}{
			"roleArn":         "arn:aws:iam::123456789012:role/TestRole",
			"roleSessionName": "test-session",
		},
		Resources: []map[string]interface{}{
			{
				"ARN":       "arn:aws:iam::123456789012:role/TestRole",
				"accountId": "123456789012",
				"type":      "AWS::IAM::Role",
			},
		},
		ResponseElements: map[string]interface{}{
			"assumedRoleUser": map[string]interface{}{
				"arn":           "arn:aws:sts::123456789012:assumed-role/TestRole/test-session",
				"assumedRoleId": "AROAXXXXXXXXXXXXXXXXX:test-session",
			},
			"credentials": map[string]interface{}{
				"accessKeyId":  "AKIAIOSFODNN7EXAMPLE",
				"expiration":   "Sep 25, 2025, 5:50:46 PM",
				"sessionToken": "FwoGZXIvYXdzEBQaD...",
			},
		},
		SharedEventID:   fmt.Sprintf("shared-%d", rand.Intn(1000000)),
		SourceIPAddress: fmt.Sprintf("192.168.%d.%d", rand.Intn(256), rand.Intn(256)),
		TlsDetails: map[string]interface{}{
			"cipherSuite":                "TLS_AES_128_GCM_SHA256",
			"clientProvidedHostHeader":   "sts.us-west-2.amazonaws.com",
			"tlsVersion":                 "TLSv1.3",
		},
		UserAgent: "aws-cli/2.13.0",
		UserIdentity: map[string]interface{}{
			"accessKeyId": "AKIAIOSFODNN7EXAMPLE",
			"accountId":   "123456789012",
			"arn":         "arn:aws:sts::123456789012:assumed-role/TestRole/test-session",
			"principalId": "AROAXXXXXXXXXXXXXXXXX:test-session",
			"type":        "AssumedRole",
		},
	}
}

func main() {
	// Load account IDs from routing table
	if err := loadAccountIDs(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading account IDs: %v\n", err)
		os.Exit(1)
	}

	if len(accountIDs) == 0 {
		fmt.Fprintf(os.Stderr, "No account IDs found in routing table\n")
		os.Exit(1)
	}

	fmt.Printf("Loaded %d account IDs from routing table\n", len(accountIDs))

	targetSize := 5 * 1024 * 1024 // 5MB target

	f, err := os.Create("test-data.ndjson.gz")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	gzWriter := gzip.NewWriter(f)
	defer gzWriter.Close()

	encoder := json.NewEncoder(gzWriter)

	count := 0
	for {
		record := generateRecord()
		if err := encoder.Encode(record); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding record: %v\n", err)
			os.Exit(1)
		}
		count++

		// Check size every 100 records
		if count % 100 == 0 {
			gzWriter.Flush()
			stat, _ := f.Stat()
			currentSize := stat.Size()

			if currentSize >= int64(targetSize) {
				break
			}

			if count % 1000 == 0 {
				fmt.Printf("Generated %d records, current size: %.2f MB\n", count, float64(currentSize)/(1024*1024))
			}
		}
	}

	fmt.Printf("Generated %d records\n", count)
	fmt.Println("Test data written to test-data.ndjson.gz")
}