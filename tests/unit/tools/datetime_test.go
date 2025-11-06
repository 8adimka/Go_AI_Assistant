package tools_test

import (
	"context"
	"testing"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/tools/datetime"
)

func TestDateTimeTool_Name(t *testing.T) {
	tool := datetime.New()
	expected := "get_today_date"

	if name := tool.Name(); name != expected {
		t.Errorf("Expected name %q, got %q", expected, name)
	}
}

func TestDateTimeTool_Description(t *testing.T) {
	tool := datetime.New()
	expected := "Get today's date and time in RFC3339 format"

	if desc := tool.Description(); desc != expected {
		t.Errorf("Expected description %q, got %q", expected, desc)
	}
}

func TestDateTimeTool_Parameters(t *testing.T) {
	tool := datetime.New()
	params := tool.Parameters()

	if params["type"] != "object" {
		t.Errorf("Expected type 'object', got %v", params["type"])
	}

	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected properties to be a map")
	}

	if len(properties) != 0 {
		t.Errorf("Expected empty properties, got %v", properties)
	}
}

func TestDateTimeTool_Execute(t *testing.T) {
	tool := datetime.New()
	ctx := context.Background()
	args := map[string]interface{}{}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Verify the result is a valid RFC3339 timestamp
	_, err = time.Parse(time.RFC3339, result)
	if err != nil {
		t.Errorf("Result is not valid RFC3339 format: %q, error: %v", result, err)
	}
}

func TestDateTimeTool_Execute_WithEmptyArgs(t *testing.T) {
	tool := datetime.New()
	ctx := context.Background()
	args := map[string]interface{}{}

	result, err := tool.Execute(ctx, args)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == "" {
		t.Error("Expected non-empty result")
	}
}
