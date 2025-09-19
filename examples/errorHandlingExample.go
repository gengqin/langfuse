package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gengqin/langfuse"
)

func main() {
	// Get configuration from environment variables
	client, err := langfuse.NewClient(langfuse.Config{
		PublicKey:   getEnvOrFail("LANGFUSE_PUBLIC_KEY"),
		SecretKey:   getEnvOrFail("LANGFUSE_SECRET_KEY"),
		BaseURL:     getEnvOrDefault("LANGFUSE_BASE_URL", "https://cloud.langfuse.com"),
		Release:     getEnvOrDefault("LANGFUSE_RELEASE", "1.0.0"),
		Environment: getEnvOrDefault("LANGFUSE_ENVIRONMENT", "development"),
		IsPublic:    false,
	})
	if err != nil {
		log.Fatal("Failed to create Langfuse client:", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := client.Close(ctx); err != nil {
			log.Printf("Failed to close client: %v", err)
		}
	}()

	ctx := context.Background()

	// Create trace for error scenario
	trace := client.CreateTrace(ctx, "failed-api-call",
		langfuse.WithTraceUserID("user-error-test"),
		langfuse.WithTraceInput("Test error handling"),
	)
	defer trace.End()

	// Create a span that will fail
	errorSpan := trace.CreateSpan("external-api-call",
		langfuse.WithSpanInput(map[string]interface{}{
			"endpoint": "https://api.example.com/data",
			"method":   "GET",
		}),
		langfuse.WithSpanLevel(langfuse.LogLevelError),
	)

	// Simulate processing
	time.Sleep(100 * time.Millisecond)

	// Set error information
	errorSpan = trace.CreateSpan("external-api-call",
		langfuse.WithSpanOutput(map[string]interface{}{
			"error":       "Connection timeout after 5s",
			"status_code": 408,
		}),
		langfuse.WithSpanMetadata(map[string]interface{}{
			"retry_count":      "3",
			"timeout_duration": "5s",
		}),
	)
	errorSpan.End()

	// Log error event
	trace.CreateEvent("api-error-occurred",
		langfuse.WithEventInput(map[string]interface{}{
			"error_type": "timeout",
			"service":    "external-api",
			"impact":     "high",
		}),
		langfuse.WithEventLevel(langfuse.LogLevelError),
	)

	// Log retry event
	trace.CreateEvent("retry-attempted",
		langfuse.WithEventInput(map[string]interface{}{
			"attempt": 1,
			"delay":   "1s",
		}),
		langfuse.WithEventLevel(langfuse.LogLevelWarning),
	)

	fmt.Println("✅ Error handling example completed")

	// Wait for traces to be sent
	time.Sleep(2 * time.Second)
}

// Helper functions
func getEnvOrFail(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Environment variable %s is required", key)
	}
	return value
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}