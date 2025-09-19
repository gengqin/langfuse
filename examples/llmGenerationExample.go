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

	// Create trace for LLM interaction
	trace := client.CreateTrace(ctx, "llm-chat",
		langfuse.WithTraceUserID("user-789"),
		langfuse.WithTraceInput("Write a Go function to sort a slice"),
	)
	defer trace.End()

	// Track LLM generation
	generation := trace.CreateGeneration("openai-completion",
		langfuse.WithGenerationModel("gpt-4o-mini"),
		langfuse.WithGenerationInput([]map[string]interface{}{
			{
				"role":    "system",
				"content": "You are a helpful Go programming assistant.",
			},
			{
				"role":    "user",
				"content": "Write a Go function to sort a slice of integers",
			},
		}),
		langfuse.WithGenerationParams(langfuse.GenerationParams{
			Temperature: floatPtr(0.7),
			MaxTokens:   intPtr(500),
			TopP:        floatPtr(1.0),
		}),
		langfuse.WithGenerationStartTime(time.Now()),
	)

	// Simulate API call
	time.Sleep(500 * time.Millisecond)

	// Set generation results using options pattern
	generation = trace.CreateGeneration("openai-completion",
		langfuse.WithGenerationOutput(map[string]interface{}{
			"content": `func sortInts(slice []int) {
	sort.Ints(slice)
}`,
			"finish_reason": "stop",
		}),
		langfuse.WithGenerationUsage(langfuse.Usage{
			PromptTokens:     45,
			CompletionTokens: 32,
			TotalTokens:      77,
		}),
		langfuse.WithGenerationCost(langfuse.Cost{
			Input:  0.00015,
			Output: 0.0006,
			Total:  0.00075,
		}),
	)
	generation.End()

	fmt.Println("✅ LLM generation example completed")

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

func floatPtr(f float64) *float64 {
	return &f
}

func intPtr(i int) *int {
	return &i
}