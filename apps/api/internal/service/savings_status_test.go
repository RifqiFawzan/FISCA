package service

import (
	"testing"
	"time"
)

// The status drives a badge in the UI, so its rules must stay explainable.
func TestSavingsStatus(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, time.November, 10, 12, 0, 0, 0, loc)
	created := now.AddDate(0, 0, -90) // goal opened 90 days ago

	cases := []struct {
		name     string
		saved    int64
		target   int64
		deadline time.Time
		created  time.Time
		want     string
	}{
		{
			name: "fresh goal with nothing saved is not behind yet",
			// Grace period: only a fraction of the window has elapsed.
			saved: 0, target: 5_000_000,
			deadline: now.AddDate(0, 0, 90), created: now.Add(-time.Minute),
			want: "on_track",
		},
		{
			name:  "progress keeps pace with the time spent",
			saved: 900_000, target: 1_000_000,
			deadline: now.AddDate(0, 0, 10), created: created,
			want: "on_track",
		},
		{
			name:  "far behind after most of the window is gone",
			saved: 100_000, target: 1_000_000,
			deadline: now.AddDate(0, 0, 10), created: created,
			want: "at_risk",
		},
		{
			name:  "target reached",
			saved: 1_000_000, target: 1_000_000,
			deadline: now.AddDate(0, 0, 10), created: created,
			want: "achieved",
		},
		{
			name:  "reached after the deadline still counts as achieved",
			saved: 2_000_000, target: 1_000_000,
			deadline: now.AddDate(0, 0, -1), created: created,
			want: "achieved",
		},
		{
			name:  "deadline passed without the target",
			saved: 500_000, target: 1_000_000,
			deadline: now.AddDate(0, 0, -1), created: created,
			want: "overdue",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := savingsStatus(tc.saved, tc.target, tc.deadline, tc.created, now)
			if got != tc.want {
				t.Fatalf("savingsStatus = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMonthsUntil(t *testing.T) {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	now := time.Date(2026, time.November, 10, 12, 0, 0, 0, loc)

	cases := []struct {
		name     string
		deadline time.Time
		want     int
	}{
		{"deadline already passed", now.AddDate(0, 0, -1), 0},
		{"deadline is now", now, 0},
		{"three weeks away still counts as a month", now.AddDate(0, 0, 21), 1},
		{"two months and a week", now.AddDate(0, 2, 7), 2},
		{"same day next month", time.Date(2026, time.December, 10, 12, 0, 0, 0, loc), 1},
		{"earlier day of a later month", time.Date(2026, time.December, 5, 12, 0, 0, 0, loc), 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := monthsUntil(now, tc.deadline); got != tc.want {
				t.Fatalf("monthsUntil = %d, want %d", got, tc.want)
			}
		})
	}
}
