package datetime

import (
	"context"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/tools/registry"
)

// DateTimeTool provides current date and time information
type DateTimeTool struct{}

// New creates a new DateTimeTool instance
func New() *DateTimeTool {
	return &DateTimeTool{}
}

// Name returns the tool name
func (d *DateTimeTool) Name() string {
	return "get_today_date"
}

// Description returns the tool description
func (d *DateTimeTool) Description() string {
	return "Get today's date and time in RFC3339 format"
}

// Parameters returns the JSON schema for parameters
func (d *DateTimeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	}
}

// Execute returns the current date and time
func (d *DateTimeTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}

// Ensure DateTimeTool implements registry.Tool interface
var _ registry.Tool = (*DateTimeTool)(nil)
