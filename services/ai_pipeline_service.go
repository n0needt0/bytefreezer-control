package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/n0needt0/go-goodies/log"
)

// AIPipelineService manages AI-assisted pipeline configuration generation
type AIPipelineService struct {
	apiKey        string
	catalogPath   string
	pluginCatalog string
	client        *http.Client
}

// NewAIPipelineService creates a new AI pipeline service
func NewAIPipelineService(apiKey, catalogPath string) *AIPipelineService {
	service := &AIPipelineService{
		apiKey:      apiKey,
		catalogPath: catalogPath,
		client: &http.Client{
			Timeout: 120 * time.Second, // Long timeout for AI generation
		},
	}

	// Load plugin catalog
	if err := service.LoadPluginCatalog(); err != nil {
		log.Warnf("Failed to load plugin catalog: %v - AI assistance will be limited", err)
	}

	return service
}

// LoadPluginCatalog loads the plugin catalog from disk
func (s *AIPipelineService) LoadPluginCatalog() error {
	data, err := os.ReadFile(s.catalogPath)
	if err != nil {
		return fmt.Errorf("failed to read catalog: %w", err)
	}

	s.pluginCatalog = string(data)
	log.Infof("Loaded plugin catalog: %d bytes", len(s.pluginCatalog))
	return nil
}

// ChatMessage represents a message in the conversation
type ChatMessage struct {
	Role    string `json:"role"`    // "user" or "assistant"
	Content string `json:"content"`
}

// GeneratePipelineRequest contains the request for pipeline generation
type GeneratePipelineRequest struct {
	TenantID      string        `json:"tenant_id"`
	DatasetID     string        `json:"dataset_id"`
	UserMessage   string        `json:"user_message"`
	DataSchema    []SchemaField `json:"data_schema,omitempty"`    // Optional: data schema for context
	SampleRecords []interface{} `json:"sample_records,omitempty"` // Optional: sample records
	History       []ChatMessage `json:"history,omitempty"`        // Optional: conversation history
}

// SchemaField represents a field in the data schema
type SchemaField struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Sample   interface{} `json:"sample,omitempty"`
	Count    int         `json:"count,omitempty"`
	Nullable bool        `json:"nullable,omitempty"`
}

// GeneratePipelineResponse contains the generated pipeline configuration
type GeneratePipelineResponse struct {
	Success         bool                     `json:"success"`
	PipelineConfig  *PipelineConfiguration   `json:"pipeline_config,omitempty"`
	Explanation     string                   `json:"explanation"`
	AssistantReply  string                   `json:"assistant_reply"`
	ValidationError string                   `json:"validation_error,omitempty"`
	Suggestions     []string                 `json:"suggestions,omitempty"`
	ConversationID  string                   `json:"conversation_id,omitempty"`
}

// PipelineConfiguration represents a transformation pipeline
type PipelineConfiguration struct {
	TenantID  string         `json:"tenant_id"`
	DatasetID string         `json:"dataset_id"`
	Enabled   bool           `json:"enabled"`
	Version   string         `json:"version"`
	Filters   []FilterConfig `json:"filters"`
}

// FilterConfig represents a single filter in the pipeline
type FilterConfig struct {
	Type      string                 `json:"type"`
	Condition string                 `json:"condition,omitempty"`
	Config    map[string]interface{} `json:"config"`
	Enabled   bool                   `json:"enabled"`
}

// ClaudeAPIRequest represents the request to Claude API
type ClaudeAPIRequest struct {
	Model       string          `json:"model"`
	MaxTokens   int             `json:"max_tokens"`
	Messages    []ClaudeMessage `json:"messages"`
	System      string          `json:"system,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
}

// ClaudeMessage represents a message in Claude API format
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeAPIResponse represents the response from Claude API
type ClaudeAPIResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model       string `json:"model"`
	StopReason  string `json:"stop_reason"`
	Usage       struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// GeneratePipeline generates a pipeline configuration based on natural language description
func (s *AIPipelineService) GeneratePipeline(ctx context.Context, req *GeneratePipelineRequest) (*GeneratePipelineResponse, error) {
	// Build system prompt with plugin catalog
	systemPrompt := s.buildSystemPrompt(req)

	// Build user message with context
	userMessage := s.buildUserMessage(req)

	// Prepare conversation history for Claude
	messages := s.buildMessageHistory(req.History, userMessage)

	// Call Claude API
	claudeResp, err := s.callClaudeAPI(ctx, systemPrompt, messages)
	if err != nil {
		return &GeneratePipelineResponse{
			Success:        false,
			AssistantReply: fmt.Sprintf("Error generating pipeline: %v", err),
		}, err
	}

	// Parse response and extract pipeline configuration
	response := s.parseClaudeResponse(claudeResp, req)

	return response, nil
}

// buildSystemPrompt creates the system prompt with plugin catalog and guidelines
func (s *AIPipelineService) buildSystemPrompt(req *GeneratePipelineRequest) string {
	var prompt strings.Builder

	prompt.WriteString("You are an expert data transformation pipeline architect for ByteFreezer. ")
	prompt.WriteString("Your role is to help users create transformation pipelines by translating ")
	prompt.WriteString("natural language descriptions into valid JSON pipeline configurations.\n\n")

	prompt.WriteString("## Your Capabilities\n\n")
	prompt.WriteString("You have access to the complete ByteFreezer Piper transformation plugin catalog ")
	prompt.WriteString("which includes 23 plugins across 6 categories:\n")
	prompt.WriteString("1. Field Manipulation (add_field, remove_field, rename_field, mutate)\n")
	prompt.WriteString("2. Pattern Matching & Parsing (grok, regex_replace, kv, date_parse)\n")
	prompt.WriteString("3. Data Enrichment (geoip, dns, useragent, fingerprint)\n")
	prompt.WriteString("4. JSON Processing (json_validate, json_flatten, uppercase_keys)\n")
	prompt.WriteString("5. Filtering & Sampling (include, exclude, drop, sample, conditional)\n")
	prompt.WriteString("6. Structural Transformation (split)\n\n")

	prompt.WriteString("## Plugin Catalog\n\n")
	prompt.WriteString(s.pluginCatalog)
	prompt.WriteString("\n\n")

	prompt.WriteString("## Your Task\n\n")
	prompt.WriteString("1. Understand the user's data transformation requirements\n")
	prompt.WriteString("2. Design an efficient pipeline using appropriate plugins\n")
	prompt.WriteString("3. Generate valid JSON configuration\n")
	prompt.WriteString("4. Provide clear explanation of what the pipeline does\n")
	prompt.WriteString("5. Suggest optimizations or alternatives when appropriate\n\n")

	prompt.WriteString("## Response Format\n\n")
	prompt.WriteString("Always respond with:\n")
	prompt.WriteString("1. A brief explanation of the pipeline strategy\n")
	prompt.WriteString("2. The JSON configuration wrapped in ```json code blocks\n")
	prompt.WriteString("3. Any important notes or warnings\n")
	prompt.WriteString("4. Suggestions for testing or improvements\n\n")

	prompt.WriteString("## Important Guidelines\n\n")
	prompt.WriteString("- Filter order matters: parse before enrich, filter early for performance\n")
	prompt.WriteString("- Use specific patterns instead of wildcards when possible\n")
	prompt.WriteString("- Always include explanatory comments for complex transformations\n")
	prompt.WriteString("- Validate JSON syntax before suggesting configurations\n")
	prompt.WriteString("- Consider error handling and edge cases\n")
	prompt.WriteString("- Keep pipelines simple and focused on user requirements\n\n")

	if len(req.DataSchema) > 0 {
		prompt.WriteString("## Current Data Schema\n\n")
		prompt.WriteString("The user's dataset has the following fields:\n")
		for _, field := range req.DataSchema {
			prompt.WriteString(fmt.Sprintf("- **%s** (%s)", field.Name, field.Type))
			if field.Sample != nil {
				prompt.WriteString(fmt.Sprintf(": Example value: `%v`", field.Sample))
			}
			prompt.WriteString("\n")
		}
		prompt.WriteString("\n")
	}

	return prompt.String()
}

// buildUserMessage constructs the user message with context
func (s *AIPipelineService) buildUserMessage(req *GeneratePipelineRequest) string {
	var msg strings.Builder

	msg.WriteString(fmt.Sprintf("**Tenant**: %s\n", req.TenantID))
	msg.WriteString(fmt.Sprintf("**Dataset**: %s\n\n", req.DatasetID))

	msg.WriteString("**User Request**:\n")
	msg.WriteString(req.UserMessage)
	msg.WriteString("\n\n")

	if len(req.SampleRecords) > 0 {
		msg.WriteString("**Sample Data** (first few records):\n")
		msg.WriteString("```json\n")
		for i, record := range req.SampleRecords {
			if i >= 3 {
				break // Only show first 3 samples
			}
			recordJSON, _ := json.MarshalIndent(record, "", "  ")
			msg.WriteString(string(recordJSON))
			msg.WriteString("\n")
		}
		msg.WriteString("```\n\n")
	}

	msg.WriteString("Please generate a transformation pipeline configuration for this request.")

	return msg.String()
}

// buildMessageHistory converts chat history to Claude format
func (s *AIPipelineService) buildMessageHistory(history []ChatMessage, newMessage string) []ClaudeMessage {
	messages := make([]ClaudeMessage, 0, len(history)+1)

	// Add conversation history
	for _, msg := range history {
		messages = append(messages, ClaudeMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// Add new user message
	messages = append(messages, ClaudeMessage{
		Role:    "user",
		Content: newMessage,
	})

	return messages
}

// callClaudeAPI makes the API call to Claude
func (s *AIPipelineService) callClaudeAPI(ctx context.Context, systemPrompt string, messages []ClaudeMessage) (*ClaudeAPIResponse, error) {
	reqBody := ClaudeAPIRequest{
		Model:       "claude-sonnet-4-5-20250929",
		MaxTokens:   4096,
		Messages:    messages,
		System:      systemPrompt,
		Temperature: 0.3, // Lower temperature for more consistent config generation
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", s.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeAPIResponse
	if err := json.Unmarshal(body, &claudeResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &claudeResp, nil
}

// parseClaudeResponse extracts pipeline config and explanation from Claude's response
func (s *AIPipelineService) parseClaudeResponse(claudeResp *ClaudeAPIResponse, req *GeneratePipelineRequest) *GeneratePipelineResponse {
	if len(claudeResp.Content) == 0 {
		return &GeneratePipelineResponse{
			Success:        false,
			AssistantReply: "No response from AI",
		}
	}

	fullText := claudeResp.Content[0].Text

	// Extract JSON from code blocks
	var config *PipelineConfiguration
	var explanation string
	var validationError string

	// Look for ```json code blocks
	jsonStart := strings.Index(fullText, "```json")
	if jsonStart != -1 {
		jsonStart += 7 // Skip ```json
		jsonEnd := strings.Index(fullText[jsonStart:], "```")
		if jsonEnd != -1 {
			jsonStr := fullText[jsonStart : jsonStart+jsonEnd]
			jsonStr = strings.TrimSpace(jsonStr)

			log.Debugf("AI extracted JSON config: %s", jsonStr)

			// Try to parse as pipeline config
			var parsed PipelineConfiguration
			if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
				validationError = fmt.Sprintf("JSON parsing error: %v", err)
				log.Warnf("AI pipeline JSON parsing failed: %v", err)
			} else {
				// Set tenant and dataset from request
				parsed.TenantID = req.TenantID
				parsed.DatasetID = req.DatasetID
				if parsed.Version == "" {
					parsed.Version = "1.0.0"
				}
				config = &parsed
				log.Infof("AI generated pipeline with %d filters", len(parsed.Filters))
			}
		} else {
			log.Warnf("AI response has ```json start but no closing ```")
		}
	} else {
		log.Warnf("AI response does not contain ```json code block")
	}

	// Extract explanation (text before JSON block)
	if jsonStart > 0 {
		explanation = strings.TrimSpace(fullText[:jsonStart])
	} else {
		explanation = fullText
	}

	// Extract suggestions (text after JSON block if any)
	var suggestions []string
	if jsonStart != -1 {
		jsonEnd := strings.Index(fullText[jsonStart:], "```")
		if jsonEnd != -1 {
			afterJSON := strings.TrimSpace(fullText[jsonStart+jsonEnd+3:])
			if afterJSON != "" {
				// Look for bullet points or numbered lists
				lines := strings.Split(afterJSON, "\n")
				for _, line := range lines {
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || strings.HasPrefix(line, "•") {
						suggestions = append(suggestions, strings.TrimSpace(line[1:]))
					}
				}
			}
		}
	}

	return &GeneratePipelineResponse{
		Success:         config != nil && validationError == "",
		PipelineConfig:  config,
		Explanation:     explanation,
		AssistantReply:  fullText,
		ValidationError: validationError,
		Suggestions:     suggestions,
	}
}

// RefreshCatalog reloads the plugin catalog from disk
func (s *AIPipelineService) RefreshCatalog() error {
	return s.LoadPluginCatalog()
}
