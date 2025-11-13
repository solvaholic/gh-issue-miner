package cmd

import (
	"testing"
	"time"
)

func TestParseTimeRange_SingleDate(t *testing.T) {
	start, end, err := parseTimeRange("2025-09-13")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start == nil || end == nil {
		t.Fatalf("expected non-nil start and end")
	}
	if !start.Equal(time.Date(2025, 9, 13, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("start mismatch: got %v", start)
	}
	if !end.Equal(time.Date(2025, 9, 14, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("end mismatch: got %v", end)
	}
}

func TestParseTimeRange_DateRange(t *testing.T) {
	start, end, err := parseTimeRange("2025-09-13..2025-09-28")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start == nil || end == nil {
		t.Fatalf("expected non-nil start and end")
	}
	if !start.Equal(time.Date(2025, 9, 13, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("start mismatch: got %v", start)
	}
	if !end.Equal(time.Date(2025, 9, 29, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("end mismatch: got %v", end)
	}
}

func TestParseTimeRange_OpenEndedLeft(t *testing.T) {
	start, end, err := parseTimeRange("..2025-09-28")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start != nil {
		t.Fatalf("expected nil start")
	}
	if end == nil {
		t.Fatalf("expected non-nil end")
	}
	if !end.Equal(time.Date(2025, 9, 29, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("end mismatch: got %v", end)
	}
}

func TestParseTimeRange_RelativeRange(t *testing.T) {
	start, end, err := parseTimeRange("60d..45d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start == nil || end == nil {
		t.Fatalf("expected non-nil start and end")
	}
	// difference should be approximately 15 days
	days := int(end.Sub(*start).Hours() / 24)
	if days < 14 || days > 16 {
		t.Fatalf("expected approximately 15 days difference, got %d", days)
	}
}

func TestParseTimeRange_Invalid(t *testing.T) {
	_, _, err := parseTimeRange("not-a-date")
	if err == nil {
		t.Fatalf("expected error for invalid input")
	}
}
