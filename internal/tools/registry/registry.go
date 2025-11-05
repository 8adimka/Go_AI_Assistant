package registry

import (
	"context"
	"log/slog"
)

// Tool defines the interface that all tools must implement
type Tool interface {
	// Name returns the unique name of the tool
	Name() string
	
	// Description returns a human-readable description of what the tool does
	Description() string
	
	// Parameters returns the JSON schema for the tool's parameters
	Parameters() map[string]interface{}
	
	// Execute runs the tool with the given arguments and returns the result
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// ToolRegistry manages the registration and retrieval of tools
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry creates a new empty tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *ToolRegistry) Register(tool Tool) {
	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		slog.Warn("Tool already registered, overwriting", "name", name)
	}
	r.tools[name] = tool
	slog.Info("Tool registered successfully", "name", name)
}

// Get returns a tool by name, or nil if not found
func (r *ToolRegistry) Get(name string) Tool {
	return r.tools[name]
}

// GetAll returns all registered tools
func (r *ToolRegistry) GetAll() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetToolNames returns the names of all registered tools
func (r *ToolRegistry) GetToolNames() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// HasTool checks if a tool with the given name is registered
func (r *ToolRegistry) HasTool(name string) bool {
	_, exists := r.tools[name]
	return exists
}

// Count returns the number of registered tools
func (r *ToolRegistry) Count() int {
	return len(r.tools)
}
