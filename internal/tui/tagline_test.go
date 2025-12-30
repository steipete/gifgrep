package tui

import (
	"testing"
	"time"
)

func TestPickTagline_OverrideIndexModulo(t *testing.T) {
	env := map[string]string{envTaglineIndex: "999"}
	getenv := func(key string) string { return env[key] }
	got := pickTagline(time.Date(2025, time.December, 26, 0, 0, 0, 0, time.UTC), getenv, func() float64 { return 0 })
	all := allTaglines()
	if len(all) == 0 {
		t.Fatalf("expected taglines")
	}
	want := all[999%len(all)]
	if got != want {
		t.Fatalf("expected modulo to land on %q, got %q", want, got)
	}
}

func TestPickTagline_UsesHolidayPoolOnHoliday(t *testing.T) {
	getenv := func(string) string { return "" }
	now := time.Date(2025, time.December, 25, 12, 0, 0, 0, time.UTC)
	got := pickTagline(now, getenv, func() float64 { return 0.5 })
	if got != taglineChristmas {
		t.Fatalf("expected christmas tagline, got %q", got)
	}
}

func TestPickTagline_UsesRegularPoolOnNormalDays(t *testing.T) {
	getenv := func(string) string { return "" }
	now := time.Date(2025, time.December, 26, 12, 0, 0, 0, time.UTC)
	got := pickTagline(now, getenv, func() float64 { return 0 })
	if got != regularTaglines[0] {
		t.Fatalf("expected first regular tagline, got %q", got)
	}
}

func TestActiveHolidayTaglines_ThanksgivingAndBlackFriday(t *testing.T) {
	thanksgiving := time.Date(2025, time.November, 27, 12, 0, 0, 0, time.UTC)
	gotThanks := activeHolidayTaglines(thanksgiving)
	if len(gotThanks) != 1 || gotThanks[0] != taglineThanksgiving {
		t.Fatalf("expected thanksgiving tagline, got %v", gotThanks)
	}

	blackFriday := time.Date(2025, time.November, 28, 12, 0, 0, 0, time.UTC)
	gotBlackFriday := activeHolidayTaglines(blackFriday)
	if len(gotBlackFriday) != 1 || gotBlackFriday[0] != taglineBlackFriday {
		t.Fatalf("expected black friday tagline, got %v", gotBlackFriday)
	}
}
