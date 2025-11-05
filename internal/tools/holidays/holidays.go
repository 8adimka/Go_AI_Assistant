package holidays

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/8adimka/Go_AI_Assistant/internal/tools/registry"
	ics "github.com/arran4/golang-ical"
)

// HolidaysTool provides holiday information from iCal calendar
type HolidaysTool struct {
	calendarURL string
}

// New creates a new HolidaysTool instance
func New(calendarURL string) *HolidaysTool {
	return &HolidaysTool{
		calendarURL: calendarURL,
	}
}

// Name returns the tool name
func (h *HolidaysTool) Name() string {
	return "get_holidays"
}

// Description returns the tool description
func (h *HolidaysTool) Description() string {
	return "Gets local bank and public holidays. Each line is a single holiday in the format 'YYYY-MM-DD: Holiday Name'."
}

// Parameters returns the JSON schema for parameters
func (h *HolidaysTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"before_date": map[string]interface{}{
				"type":        "string",
				"description": "Optional date in RFC3339 format to get holidays before this date. If not provided, all holidays will be returned.",
			},
			"after_date": map[string]interface{}{
				"type":        "string",
				"description": "Optional date in RFC3339 format to get holidays after this date. If not provided, all holidays will be returned.",
			},
			"max_count": map[string]interface{}{
				"type":        "integer",
				"description": "Optional maximum number of holidays to return. If not provided, all holidays will be returned.",
			},
		},
	}
}

// Execute loads and filters holidays based on provided arguments
func (h *HolidaysTool) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	slog.InfoContext(ctx, "Loading holidays", "calendar_url", h.calendarURL)

	events, err := h.loadCalendar(ctx, h.calendarURL)
	if err != nil {
		return "", err
	}

	var beforeDate, afterDate time.Time
	var maxCount int

	// Parse optional arguments
	if beforeDateStr, ok := args["before_date"].(string); ok && beforeDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, beforeDateStr); err == nil {
			beforeDate = parsed
		}
	}

	if afterDateStr, ok := args["after_date"].(string); ok && afterDateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, afterDateStr); err == nil {
			afterDate = parsed
		}
	}

	if maxCountVal, ok := args["max_count"].(json.Number); ok {
		if count, err := maxCountVal.Int64(); err == nil {
			maxCount = int(count)
		}
	}

	var holidays []string
	for _, event := range events {
		date, err := event.GetAllDayStartAt()
		if err != nil {
			continue
		}

		// Apply filters
		if maxCount > 0 && len(holidays) >= maxCount {
			break
		}

		if !beforeDate.IsZero() && date.After(beforeDate) {
			continue
		}

		if !afterDate.IsZero() && date.Before(afterDate) {
			continue
		}

		holidayName := event.GetProperty(ics.ComponentPropertySummary).Value
		holidays = append(holidays, date.Format(time.DateOnly)+": "+holidayName)
	}

	return strings.Join(holidays, "\n"), nil
}

// loadCalendar loads holiday events from iCal URL
func (h *HolidaysTool) loadCalendar(ctx context.Context, url string) ([]*ics.VEvent, error) {
	// Use environment variable if available, otherwise use default
	calendarURL := url
	if envURL := os.Getenv("HOLIDAY_CALENDAR_LINK"); envURL != "" {
		calendarURL = envURL
	}

	cal, err := ics.ParseCalendarFromUrl(calendarURL)
	if err != nil {
		return nil, err
	}

	var events []*ics.VEvent
	for _, component := range cal.Components {
		if event, ok := component.(*ics.VEvent); ok {
			events = append(events, event)
		}
	}

	slog.InfoContext(ctx, "Loaded holiday events", "count", len(events))
	return events, nil
}

// Ensure HolidaysTool implements registry.Tool interface
var _ registry.Tool = (*HolidaysTool)(nil)
