package scheduler

import (
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestNextRunForManualSchedule(t *testing.T) {
	t.Parallel()

	scheduler := New()
	next, err := scheduler.NextRun(domain.Schedule{Type: domain.ScheduleTypeManual}, time.Now())
	if err != nil {
		t.Fatalf("NextRun: %v", err)
	}
	if next != nil {
		t.Fatalf("manual next run: got %v want nil", next)
	}
}

func TestNextRunForCronSchedule(t *testing.T) {
	t.Parallel()

	scheduler := New()
	from := time.Date(2026, 5, 7, 8, 0, 0, 0, time.UTC)
	next, err := scheduler.NextRun(domain.Schedule{
		Type:       domain.ScheduleTypeCron,
		Expression: "0 9 * * MON-FRI",
	}, from)
	if err != nil {
		t.Fatalf("NextRun: %v", err)
	}
	want := time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC)
	if next == nil || !next.Equal(want) {
		t.Fatalf("cron next run: got %v want %v", next, want)
	}
}

func TestNextRunRejectsInvalidCron(t *testing.T) {
	t.Parallel()

	scheduler := New()
	_, err := scheduler.NextRun(domain.Schedule{
		Type:       domain.ScheduleTypeCron,
		Expression: "not cron",
	}, time.Now())
	if err == nil {
		t.Fatal("NextRun returned nil error")
	}
}
