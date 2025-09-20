# Langfuse Go SDK

A Go SDK for [Langfuse](https://langfuse.com), the open-source LLM observability platform. This SDK enables you to track and monitor your LLM applications with comprehensive tracing, generation tracking, and event logging capabilities.

## Features

- 🔍 **Comprehensive Tracing**: Track complex LLM workflows with nested spans
- 🤖 **LLM Generation Tracking**: Monitor model usage, costs, and performance metrics
- 📊 **Event Logging**: Log important events with different severity levels
- 🔐 **Secure Authentication**: Built-in support for Langfuse API keys
- 🌐 **OpenTelemetry Integration**: Built on industry-standard OpenTelemetry
- 📈 **Usage & Cost Tracking**: Monitor token usage and associated costs

## Installation

```bash
go get github.com/qinrichard/langfuse
```

## Quick Start

### 1. Set Environment Variables

```bash
export LANGFUSE_PUBLIC_KEY="pk-lf-your-public-key"
export LANGFUSE_SECRET_KEY="sk-lf-your-secret-key"
export LANGFUSE_BASE_URL="https://cloud.langfuse.com"  # optional
```

### 2. Basic Usage

```go
package main

import (
    "context"
    "os"
    "github.com/qinrichard/langfuse"
)

func main() {
    // Initialize client
    client, err := langfuse.NewClient(langfuse.Config{
        PublicKey: os.Getenv("LANGFUSE_PUBLIC_KEY"),
        SecretKey: os.Getenv("LANGFUSE_SECRET_KEY"),
    })
    if err != nil {
        panic(err)
    }
    defer client.Close(context.Background())

    // Create a trace
    trace := client.CreateTrace(context.Background(), "my-llm-app",
        langfuse.WithTraceUserID("user-123"),
        langfuse.WithTraceInput("Hello, world!"),
    )
    defer trace.End()

    // Track LLM generation
    generation := trace.CreateGeneration("openai-call",
        langfuse.WithGenerationModel("gpt-4o-mini"),
        langfuse.WithGenerationUsage(langfuse.Usage{
            PromptTokens:     10,
            CompletionTokens: 20,
            TotalTokens:      30,
        }),
    )
    generation.End()
}
```

## Configuration

### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `LANGFUSE_PUBLIC_KEY` | Your Langfuse public key | Yes | - |
| `LANGFUSE_SECRET_KEY` | Your Langfuse secret key | Yes | - |
| `LANGFUSE_BASE_URL` | Langfuse instance URL | No | `https://cloud.langfuse.com` |
| `LANGFUSE_RELEASE` | Application release version | No | - |
| `LANGFUSE_ENVIRONMENT` | Environment (dev, staging, prod) | No | - |

### Client Configuration

```go
client, err := langfuse.NewClient(langfuse.Config{
    PublicKey:   "pk-lf-...",
    SecretKey:   "sk-lf-...",
    BaseURL:     "https://cloud.langfuse.com", // optional
    Release:     "1.0.0",                      // optional
    Environment: "production",                 // optional
    IsPublic:    false,                        // optional
})
```

## Core Concepts

### 1. Traces

Traces represent the top-level execution context of your LLM application:

```go
trace := client.CreateTrace(ctx, "user-query",
    langfuse.WithTraceUserID("user-123"),
    langfuse.WithTraceSessionID("session-456"),
    langfuse.WithTraceTags([]string{"customer-support", "urgent"}),
    langfuse.WithTraceMetadata(map[string]interface{}{
        "user_tier": "premium",
        "channel":   "web",
    }),
    langfuse.WithTraceInput("How do I use this SDK?"),
)
defer trace.End()
```

### 2. Spans

Spans track individual operations within a trace:

```go
span := trace.CreateSpan("data-processing",
    langfuse.WithSpanInput("Raw user data"),
    langfuse.WithSpanMetadata(map[string]interface{}{
        "processing_type": "nlp",
    }),
    langfuse.WithSpanLevel(langfuse.LogLevelDefault),
)
// ... do work ...
span.End()
```

### 3. Generations

Generations specifically track LLM API calls:

```go
generation := trace.CreateGeneration("openai-completion",
    langfuse.WithGenerationModel("gpt-4o-mini"),
    langfuse.WithGenerationInput([]map[string]interface{}{
        {"role": "user", "content": "Write a function"},
    }),
    langfuse.WithGenerationParams(langfuse.GenerationParams{
        Temperature: floatPtr(0.7),
        MaxTokens:   intPtr(500),
    }),
    langfuse.WithGenerationUsage(langfuse.Usage{
        PromptTokens:     45,
        CompletionTokens: 32,
        TotalTokens:      77,
    }),
    langfuse.WithGenerationCost(langfuse.Cost{
        Total: 0.00075,
    }),
)
generation.End()
```

### 4. Events

Events log point-in-time occurrences:

```go
trace.CreateEvent("error-occurred",
    langfuse.WithEventInput(map[string]interface{}{
        "error_type": "timeout",
        "service":    "external-api",
    }),
    langfuse.WithEventLevel(langfuse.LogLevelError),
)
```

## Examples

The `examples/` directory contains complete, runnable examples:

- **[simpleTraceExample.go](examples/simpleTraceExample.go)**: Basic trace with span
- **[llmGenerationExample.go](examples/llmGenerationExample.go)**: LLM generation tracking
- **[complexTraceExample.go](examples/complexTraceExample.go)**: Multi-step workflow
- **[errorHandlingExample.go](examples/errorHandlingExample.go)**: Error scenarios

### Running Examples

```bash
# Set your environment variables first
export LANGFUSE_PUBLIC_KEY="pk-lf-your-key"
export LANGFUSE_SECRET_KEY="sk-lf-your-key"

# Run individual examples
go run examples/simpleTraceExample.go
go run examples/llmGenerationExample.go
go run examples/complexTraceExample.go
go run examples/errorHandlingExample.go
```

## API Reference

### Log Levels

```go
langfuse.LogLevelDebug   // Debug information
langfuse.LogLevelDefault // Normal operations
langfuse.LogLevelWarning // Warning conditions
langfuse.LogLevelError   // Error conditions
```

### Usage Tracking

```go
type Usage struct {
    PromptTokens     int `json:"prompt_tokens,omitempty"`
    CompletionTokens int `json:"completion_tokens,omitempty"`
    TotalTokens      int `json:"total_tokens,omitempty"`
}
```

### Cost Tracking

```go
type Cost struct {
    Total  float64 `json:"total,omitempty"`
    Input  float64 `json:"input,omitempty"`
    Output float64 `json:"output,omitempty"`
}
```

### Generation Parameters

```go
type GenerationParams struct {
    Temperature      *float64           `json:"temperature,omitempty"`
    MaxTokens        *int               `json:"max_tokens,omitempty"`
    TopP             *float64           `json:"top_p,omitempty"`
    FrequencyPenalty *float64           `json:"frequency_penalty,omitempty"`
    PresencePenalty  *float64           `json:"presence_penalty,omitempty"`
    Stop             []string           `json:"stop,omitempty"`
}
```

## Best Practices

1. **Always close the client**: Use `defer client.Close(ctx)` to ensure proper cleanup
2. **End traces and spans**: Use `defer trace.End()` and `defer span.End()`
3. **Use environment variables**: Never hardcode API keys in your code
4. **Add context**: Include relevant metadata, user IDs, and session IDs
5. **Handle errors**: Check for errors when creating the client
6. **Use appropriate log levels**: Choose the right severity for events

## Integration with Popular LLM Libraries

### OpenAI

```go
// Before making OpenAI call
generation := trace.CreateGeneration("openai-completion",
    langfuse.WithGenerationModel("gpt-4o-mini"),
    langfuse.WithGenerationInput(messages),
)

// Make your OpenAI API call
// response, err := openaiClient.CreateCompletion(...)

// Log the results
generation.span.SetAttributes(
    langfuse.WithGenerationUsage(langfuse.Usage{
        PromptTokens:     response.Usage.PromptTokens,
        CompletionTokens: response.Usage.CompletionTokens,
        TotalTokens:      response.Usage.TotalTokens,
    }),
    langfuse.WithGenerationOutput(response.Choices[0].Message),
)
generation.End()
```

## Troubleshooting

### Common Issues

1. **Import errors**: Make sure to run `go mod tidy` after installation
2. **Authentication failures**: Verify your API keys are correct and have proper permissions
3. **Network issues**: Check if your firewall allows HTTPS traffic to Langfuse

### Debug Mode

Set environment variable for verbose logging:
```bash
export LANGFUSE_DEBUG=true
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `go test ./...`
6. Submit a pull request

## License

[MIT License](LICENSE)

## Support

- 📖 [Langfuse Documentation](https://langfuse.com/docs)
- 💬 [Discord Community](https://discord.gg/7NXusRtqYU)
- 🐛 [Issue Tracker](https://github.com/qinrichard/langfuse/issues)