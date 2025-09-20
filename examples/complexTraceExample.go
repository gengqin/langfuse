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

	// Create trace for a complex workflow
	trace := client.CreateTrace(ctx, "code-review-workflow",
		langfuse.WithTraceUserID("developer-456"),
		langfuse.WithTraceSessionID("review-session-123"),
		langfuse.WithTraceTags([]string{"code-review", "golang", "automated"}),
		langfuse.WithTraceInput(map[string]interface{}{
			"repository":    "my-go-project",
			"pr_number":     42,
			"files_changed": []string{"main.go", "handler.go"},
		}),
	)
	defer trace.End()

	// Step 1: Code analysis span
	analysisSpan := trace.CreateSpan("code-analysis",
		langfuse.WithSpanInput(map[string]interface{}{
			"files":         []string{"main.go", "handler.go"},
			"analysis_type": "static",
		}),
	)
	time.Sleep(200 * time.Millisecond)

	// Simulate analysis processing
	analysisSpan.End()

	// Step 2: LLM-based code review
	reviewGeneration := trace.CreateGeneration("code-review-llm",
		langfuse.WithGenerationModel("claude-3-5-sonnet-20241022"),
		langfuse.WithGenerationInput(map[string]interface{}{
			"role":    "code_reviewer",
			"code":    "package main\n\nfunc main() { ... }",
			"context": "Go web service code review",
		}),
		langfuse.WithGenerationUsage(langfuse.Usage{
			PromptTokens:     1250,
			CompletionTokens: 340,
			TotalTokens:      1590,
		}),
	)
	time.Sleep(800 * time.Millisecond)
	reviewGeneration.End()

	// Step 3: Log important events
	trace.CreateEvent("review-completed",
		langfuse.WithEventInput(map[string]interface{}{
			"review_score":  8.5,
			"issues_found":  3,
			"auto_approved": true,
		}),
		langfuse.WithEventLevel(langfuse.LogLevelDefault),
		langfuse.WithEventMetadata(map[string]interface{}{
			"reviewer":  "ai-assistant",
			"timestamp": time.Now().Unix(),
		}),
	)

	// Trace will automatically end with the defer statement

	fmt.Println("✅ Complex trace example completed")

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