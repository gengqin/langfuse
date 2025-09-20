// Package langfuse provides a Go SDK for Langfuse observability platform
package langfuse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
)

// Client represents a Langfuse client
type Client struct {
	tracer       oteltrace.Tracer
	provider     *trace.TracerProvider
	publicKey    string
	secretKey    string
	baseURL      string
	release      string
	environment  string
	isPublic     bool
}

// Config holds configuration for Langfuse client
type Config struct {
	PublicKey   string
	SecretKey   string
	BaseURL     string // Optional, defaults to https://cloud.langfuse.com
	Release     string // Optional
	Environment string // Optional
	IsPublic    bool   // Optional, defaults to false
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens,omitempty"`
	CompletionTokens int `json:"completion_tokens,omitempty"`
	TotalTokens      int `json:"total_tokens,omitempty"`
}

// Cost represents cost information
type Cost struct {
	Total  float64 `json:"total,omitempty"`
	Input  float64 `json:"input,omitempty"`
	Output float64 `json:"output,omitempty"`
}

// GenerationParams represents parameters for LLM generation
type GenerationParams struct {
	Temperature      *float64           `json:"temperature,omitempty"`
	MaxTokens        *int               `json:"max_tokens,omitempty"`
	TopP             *float64           `json:"top_p,omitempty"`
	FrequencyPenalty *float64           `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64           `json:"presence_penalty,omitempty"`
	Stop             []string           `json:"stop,omitempty"`
	Other            map[string]interface{} `json:"-"`
}

// ObservationType represents the type of observation
type ObservationType string

const (
	ObservationTypeSpan       ObservationType = "span"
	ObservationTypeGeneration ObservationType = "generation"
	ObservationTypeEvent      ObservationType = "event"
)

// LogLevel represents the severity level of an observation
type LogLevel string

const (
	LogLevelDebug   LogLevel = "DEBUG"
	LogLevelDefault LogLevel = "DEFAULT"
	LogLevelWarning LogLevel = "WARNING"
	LogLevelError   LogLevel = "ERROR"
)

// NewClient creates a new Langfuse client
func NewClient(config Config) (*Client, error) {
	if config.PublicKey == "" || config.SecretKey == "" {
		return nil, fmt.Errorf("public key and secret key are required")
	}

	if config.BaseURL == "" {
		config.BaseURL = "https://cloud.langfuse.com"
	}

	// Create OTLP exporter with proper URL handling
	baseURL := config.BaseURL
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}
	
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	
	fmt.Printf("Debug - baseURL: %s\n", baseURL)
	fmt.Printf("Debug - u.Scheme: %s\n", u.Scheme)
	fmt.Printf("Debug - u.Host: %s\n", u.Host)

	// Use host only for endpoint when using WithURLPath
	endpoint := u.Host
	fmt.Printf("Debug - constructed endpoint: %s\n", endpoint)

	authHeader := fmt.Sprintf("Basic %s", encodeBasicAuth(config.PublicKey, config.SecretKey))

	// Build options slice for cleaner conditional logic
	options := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithURLPath("/api/public/otel/v1/traces"),
		otlptracehttp.WithHeaders(map[string]string{
			"Authorization": authHeader,
		}),
	}

	// Add scheme-specific options
	if u.Scheme == "http" {
		options = append(options, otlptracehttp.WithInsecure())
	}

	// Create exporter with all options
	exporter, err := otlptracehttp.New(context.Background(), options...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create resource
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName("langfuse-go-sdk"),
	)

	// Create trace provider
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(provider)

	client := &Client{
		tracer:      provider.Tracer("langfuse-go-sdk"),
		provider:    provider,
		publicKey:   config.PublicKey,
		secretKey:   config.SecretKey,
		baseURL:     config.BaseURL,
		release:     config.Release,
		environment: config.Environment,
		isPublic:    config.IsPublic,
	}

	return client, nil
}

// Close gracefully shuts down the client
func (c *Client) Close(ctx context.Context) error {
	return c.provider.Shutdown(ctx)
}

// Trace represents a Langfuse trace
type Trace struct {
	client  *Client
	ctx     context.Context
	span    oteltrace.Span
	traceID string
}

// CreateTrace creates a new trace
func (c *Client) CreateTrace(ctx context.Context, name string, opts ...TraceOption) *Trace {
	spanCtx, span := c.tracer.Start(ctx, name)

	// Set trace-level attributes
	attrs := []attribute.KeyValue{}
	if c.release != "" {
		attrs = append(attrs, attribute.String("langfuse.release", c.release))
	}
	if c.environment != "" {
		attrs = append(attrs, attribute.String("langfuse.environment", c.environment))
	}
	if c.isPublic {
		attrs = append(attrs, attribute.Bool("langfuse.trace.public", c.isPublic))
	}

	span.SetAttributes(attrs...)

	trace := &Trace{
		client:  c,
		ctx:     spanCtx,
		span:    span,
		traceID: span.SpanContext().TraceID().String(),
	}

	// Apply options
	for _, opt := range opts {
		opt(trace)
	}

	return trace
}

// TraceOption defines options for trace creation
type TraceOption func(*Trace)

// WithTraceUserID sets the user ID for the trace
func WithTraceUserID(userID string) TraceOption {
	return func(t *Trace) {
		t.span.SetAttributes(attribute.String("langfuse.user.id", userID))
	}
}

// WithTraceSessionID sets the session ID for the trace
func WithTraceSessionID(sessionID string) TraceOption {
	return func(t *Trace) {
		t.span.SetAttributes(attribute.String("langfuse.session.id", sessionID))
	}
}

// WithTraceTags sets tags for the trace
func WithTraceTags(tags []string) TraceOption {
	return func(t *Trace) {
		tagsJSON, _ := json.Marshal(tags)
		t.span.SetAttributes(attribute.String("langfuse.trace.tags", string(tagsJSON)))
	}
}

// WithTraceMetadata sets metadata for the trace
func WithTraceMetadata(metadata map[string]interface{}) TraceOption {
	return func(t *Trace) {
		for key, value := range metadata {
			if str, ok := value.(string); ok {
				t.span.SetAttributes(attribute.String(fmt.Sprintf("langfuse.trace.metadata.%s", key), str))
			}
		}
	}
}

// WithTraceInput sets the input for the trace
func WithTraceInput(input interface{}) TraceOption {
	return func(t *Trace) {
		inputJSON, _ := json.Marshal(input)
		t.span.SetAttributes(attribute.String("langfuse.trace.input", string(inputJSON)))
	}
}

// WithTraceOutput sets the output for the trace
func WithTraceOutput(output interface{}) TraceOption {
	return func(t *Trace) {
		outputJSON, _ := json.Marshal(output)
		t.span.SetAttributes(attribute.String("langfuse.trace.output", string(outputJSON)))
	}
}

// End ends the trace
func (t *Trace) End() {
	t.span.End()
}

// Span represents a Langfuse span observation
type Span struct {
	trace *Trace
	span  oteltrace.Span
	ctx   context.Context
}

// SpanOption defines options for span creation
type SpanOption func(*Span)

// WithSpanMetadata sets metadata for the span
func WithSpanMetadata(metadata map[string]interface{}) SpanOption {
	return func(s *Span) {
		for key, value := range metadata {
			if str, ok := value.(string); ok {
				s.span.SetAttributes(attribute.String(fmt.Sprintf("langfuse.observation.metadata.%s", key), str))
			}
		}
	}
}

// WithSpanInput sets the input for the span
func WithSpanInput(input interface{}) SpanOption {
	return func(s *Span) {
		inputJSON, _ := json.Marshal(input)
		s.span.SetAttributes(attribute.String("langfuse.observation.input", string(inputJSON)))
	}
}

// WithSpanOutput sets the output for the span
func WithSpanOutput(output interface{}) SpanOption {
	return func(s *Span) {
		outputJSON, _ := json.Marshal(output)
		s.span.SetAttributes(attribute.String("langfuse.observation.output", string(outputJSON)))
	}
}

// WithSpanLevel sets the log level for the span
func WithSpanLevel(level LogLevel) SpanOption {
	return func(s *Span) {
		s.span.SetAttributes(attribute.String("langfuse.observation.level", string(level)))
		
		// Also set OpenTelemetry status based on level
		switch level {
		case LogLevelError:
			s.span.SetStatus(codes.Error, "")
		case LogLevelWarning:
			s.span.SetStatus(codes.Error, "")
		default:
			s.span.SetStatus(codes.Ok, "")
		}
	}
}

// CreateSpan creates a new span within the trace
func (t *Trace) CreateSpan(name string, opts ...SpanOption) *Span {
	ctx, span := t.client.tracer.Start(t.ctx, name)
	
	// Set span type
	span.SetAttributes(attribute.String("langfuse.observation.type", string(ObservationTypeSpan)))

	s := &Span{
		trace: t,
		span:  span,
		ctx:   ctx,
	}

	// Apply options
	for _, opt := range opts {
		opt(s)
	}

	return s
}

// End ends the span
func (s *Span) End() {
	s.span.End()
}

// Generation represents a Langfuse generation observation
type Generation struct {
	trace *Trace
	span  oteltrace.Span
	ctx   context.Context
}

// GenerationOption defines options for generation creation
type GenerationOption func(*Generation)

// WithGenerationModel sets the model for the generation
func WithGenerationModel(model string) GenerationOption {
	return func(g *Generation) {
		g.span.SetAttributes(attribute.String("langfuse.observation.model.name", model))
	}
}

// WithGenerationUsage sets the usage for the generation
func WithGenerationUsage(usage Usage) GenerationOption {
	return func(g *Generation) {
		usageJSON, _ := json.Marshal(usage)
		g.span.SetAttributes(attribute.String("langfuse.observation.usage_details", string(usageJSON)))
	}
}

// WithGenerationCost sets the cost for the generation
func WithGenerationCost(cost Cost) GenerationOption {
	return func(g *Generation) {
		costJSON, _ := json.Marshal(cost)
		g.span.SetAttributes(attribute.String("langfuse.observation.cost_details", string(costJSON)))
	}
}

// WithGenerationParams sets the parameters for the generation
func WithGenerationParams(params GenerationParams) GenerationOption {
	return func(g *Generation) {
		paramsJSON, _ := json.Marshal(params)
		g.span.SetAttributes(attribute.String("langfuse.observation.model.parameters", string(paramsJSON)))
	}
}

// WithGenerationInput sets the input for the generation
func WithGenerationInput(input interface{}) GenerationOption {
	return func(g *Generation) {
		inputJSON, _ := json.Marshal(input)
		g.span.SetAttributes(attribute.String("langfuse.observation.input", string(inputJSON)))
	}
}

// WithGenerationOutput sets the output for the generation
func WithGenerationOutput(output interface{}) GenerationOption {
	return func(g *Generation) {
		outputJSON, _ := json.Marshal(output)
		g.span.SetAttributes(attribute.String("langfuse.observation.output", string(outputJSON)))
	}
}

// WithGenerationStartTime sets the completion start time for the generation
func WithGenerationStartTime(startTime time.Time) GenerationOption {
	return func(g *Generation) {
		g.span.SetAttributes(attribute.String("langfuse.observation.completion_start_time", startTime.Format(time.RFC3339)))
	}
}

// WithGenerationPrompt sets the prompt name and version for the generation
func WithGenerationPrompt(name string, version int) GenerationOption {
	return func(g *Generation) {
		g.span.SetAttributes(
			attribute.String("langfuse.observation.prompt.name", name),
			attribute.Int("langfuse.observation.prompt.version", version),
		)
	}
}

// CreateGeneration creates a new generation within the trace
func (t *Trace) CreateGeneration(name string, opts ...GenerationOption) *Generation {
	ctx, span := t.client.tracer.Start(t.ctx, name)
	
	// Set generation type
	span.SetAttributes(attribute.String("langfuse.observation.type", string(ObservationTypeGeneration)))

	g := &Generation{
		trace: t,
		span:  span,
		ctx:   ctx,
	}

	// Apply options
	for _, opt := range opts {
		opt(g)
	}

	return g
}

// End ends the generation
func (g *Generation) End() {
	g.span.End()
}

// Event represents a Langfuse event observation
type Event struct {
	trace *Trace
	span  oteltrace.Span
}

// EventOption defines options for event creation
type EventOption func(*Event)

// WithEventMetadata sets metadata for the event
func WithEventMetadata(metadata map[string]interface{}) EventOption {
	return func(e *Event) {
		for key, value := range metadata {
			if str, ok := value.(string); ok {
				e.span.SetAttributes(attribute.String(fmt.Sprintf("langfuse.observation.metadata.%s", key), str))
			}
		}
	}
}

// WithEventInput sets the input for the event
func WithEventInput(input interface{}) EventOption {
	return func(e *Event) {
		inputJSON, _ := json.Marshal(input)
		e.span.SetAttributes(attribute.String("langfuse.observation.input", string(inputJSON)))
	}
}

// WithEventLevel sets the log level for the event
func WithEventLevel(level LogLevel) EventOption {
	return func(e *Event) {
		e.span.SetAttributes(attribute.String("langfuse.observation.level", string(level)))
		
		// Also set OpenTelemetry status based on level
		switch level {
		case LogLevelError:
			e.span.SetStatus(codes.Error, "")
		case LogLevelWarning:
			e.span.SetStatus(codes.Error, "")
		default:
			e.span.SetStatus(codes.Ok, "")
		}
	}
}

// CreateEvent creates a new event within the trace
func (t *Trace) CreateEvent(name string, opts ...EventOption) *Event {
	_, span := t.client.tracer.Start(t.ctx, name)
	
	// Set event type and immediately end it (events are instantaneous)
	span.SetAttributes(attribute.String("langfuse.observation.type", string(ObservationTypeEvent)))

	e := &Event{
		trace: t,
		span:  span,
	}

	// Apply options
	for _, opt := range opts {
		opt(e)
	}

	// Events are instantaneous, so we end them immediately
	span.End()

	return e
}

// Utility function to encode basic auth
func encodeBasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64Encode([]byte(auth))
}

// Simple base64 encoding function
func base64Encode(data []byte) string {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result strings.Builder
	
	for i := 0; i < len(data); i += 3 {
		var b1, b2, b3 byte
		b1 = data[i]
		if i+1 < len(data) {
			b2 = data[i+1]
		}
		if i+2 < len(data) {
			b3 = data[i+2]
		}
		
		result.WriteByte(base64Chars[(b1>>2)&0x3F])
		result.WriteByte(base64Chars[((b1&0x03)<<4)|((b2>>4)&0x0F)])
		
		if i+1 < len(data) {
			result.WriteByte(base64Chars[((b2&0x0F)<<2)|((b3>>6)&0x03)])
		} else {
			result.WriteByte('=')
		}
		
		if i+2 < len(data) {
			result.WriteByte(base64Chars[b3&0x3F])
		} else {
			result.WriteByte('=')
		}
	}
	
	return result.String()
}

