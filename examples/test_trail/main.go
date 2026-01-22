package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aizacoders/gotrails/gotrails"
	"github.com/aizacoders/gotrails/sink"
)

func main() {
	// Create a simple trail
	cfg := gotrails.NewConfig(
		gotrails.WithServiceName("test-service"),
		gotrails.WithEnvironment("development"),
	)

	trail := gotrails.NewTrail("trace-123", "req-456", cfg)

	// Set request
	trail.SetRequest(&gotrails.HTTPRequest{
		Method: "POST",
		Path:   "/v1/payments",
		Headers: map[string][]string{
			"content-type": []string{"application/json"},
		},
		Body: map[string]any{
			"amount":   150000,
			"password": "secret123",
		},
	})

	// Simulate processing time
	time.Sleep(10 * time.Millisecond)

	// Set response
	trail.SetResponse(&gotrails.HTTPResponse{
		Status: 201,
		Body: map[string]any{
			"payment_id": "pay-789",
			"status":     "PENDING",
		},
	})

	// Set metadata
	trail.SetMetadata("user_id", "u-123")

	// Finalize
	trail.Finalize()

	// Test JSON marshal directly
	fmt.Println("=== Direct JSON Marshal ===")
	data, err := json.MarshalIndent(trail, "", "  ")
	if err != nil {
		fmt.Printf("Marshal error: %v\n", err)
	} else {
		fmt.Println(string(data))
	}

	// Test with StdoutSink
	fmt.Println("\n=== StdoutSink Output ===")
	stdoutSink := sink.NewStdoutSink(sink.WithPrettyPrint(true))
	err = stdoutSink.Write(context.Background(), trail)
	if err != nil {
		fmt.Printf("Sink error: %v\n", err)
	}
}
