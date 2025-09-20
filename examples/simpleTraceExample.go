package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/qinrichard/langfuse"
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

	// Create a trace with metadata
	trace := client.CreateTrace(ctx, "user-query",
		langfuse.WithTraceUserID("user-123"),
		langfuse.WithTraceSessionID("session-456"),
		langfuse.WithTraceTags([]string{"customer-support", "urgent"}),
		langfuse.WithTraceMetadata(map[string]interface{}{
			"user_tier":  "premium",
			"query_type": "technical",
			"channel":    "web",
		}),
		langfuse.WithTraceInput(map[string]interface{}{
			"query":   "How do I integrate Langfuse with Go?",
			"context": "User is trying to set up observability",
		}),
	)
	defer trace.End()

	// Create a span for processing
	span := trace.CreateSpan("process-query",
		langfuse.WithSpanInput("How do I integrate Langfuse with Go?"),
		langfuse.WithSpanMetadata(map[string]interface{}{
			"processing_type": "nlp",
			"language":        "en",
		}),
	)

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Set span output using WithSpanOutput option
	span = trace.CreateSpan("process-query",
		langfuse.WithSpanOutput("Processed query for Go integration help"),
	)
	span.End()

	// Set trace output using WithTraceOutput option
	trace = client.CreateTrace(ctx, "user-query",
		langfuse.WithTraceOutput(map[string]interface{}{
			"response":   "Here's how to integrate Langfuse with Go...",
			"confidence": 0.95,
		}),
	)

	fmt.Println("✅ Simple trace example completed")

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