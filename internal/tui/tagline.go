package tui

import (
	"math/rand"
	"strconv"
	"time"
)

const (
	taglineDefault = "Grep the GIF. Stick the landing."

	taglineNewYear      = "New year, new reactions. Same terminal."
	taglineValentine    = "Valentine's Day: be mine. Or at least be looped."
	taglineAprilFools   = "April Fools': trust no GIF. Especially this one."
	taglineStarWars     = "May the GIF be with you."
	taglinePrideMonth   = "Pride Month: full color, zero red in CI."
	taglineIndependence = "July 4th: fireworks outside, not in your stack traces."
	taglineHalloween    = "Halloween: ship treats, not trick exceptions."
	taglineThanksgiving = "Thanksgiving: grateful for green builds and better GIFs."
	taglineBlackFriday  = "Black Friday: no deals, just premium reactions."
	taglineChristmas    = "Christmas: all is calm, all is animated."
	envTaglineIndex     = "GIFGREP_TAGLINE_INDEX"
)

var regularTaglines = []string{
	taglineDefault,
	"Zero context, maximum reaction.",
	"Search. Select. Send. Regret nothing.",
	"Your command line, now with subtext.",
	"The fastest way to say “same”.",
	"Terminal GIFs: because meetings have feelings.",
	"Pipes welcome. Vibes optional.",
	"When words fail, grep harder.",
	"Search like you mean it: bytes, vibes, boom.",
	"Animated reactions, zero tab clutter.",
	"Your terminal’s GIF aisle. No checkout lines.",
	"Arrow keys in, serotonin out.",
	"Find the bit. Drop the mic. Paste the GIF.",
	"Less scrolling, more scene-stealing.",
}

var holidayTaglines = []string{
	taglineNewYear,
	taglineValentine,
	taglineAprilFools,
	taglineStarWars,
	taglinePrideMonth,
	taglineIndependence,
	taglineHalloween,
	taglineThanksgiving,
	taglineBlackFriday,
	taglineChristmas,
}

type holidayRule func(time.Time) bool

var holidayRules = map[string]holidayRule{
	taglineNewYear:      onMonthDay(time.January, 1),
	taglineValentine:    onMonthDay(time.February, 14),
	taglineAprilFools:   onMonthDay(time.April, 1),
	taglineStarWars:     onMonthDay(time.May, 4),
	taglinePrideMonth:   inMonth(time.June),
	taglineIndependence: onMonthDay(time.July, 4),
	taglineHalloween:    onMonthDay(time.October, 31),
	taglineThanksgiving: isFourthThursdayOfNovember,
	taglineBlackFriday:  isDayAfter(isFourthThursdayOfNovember),
	taglineChristmas:    onMonthDay(time.December, 25),
}

var taglineRand = rand.New(rand.NewSource(time.Now().UnixNano()))

func onMonthDay(month time.Month, day int) holidayRule {
	return func(date time.Time) bool {
		d := date.UTC()
		return d.Month() == month && d.Day() == day
	}
}

func inMonth(month time.Month) holidayRule {
	return func(date time.Time) bool {
		return date.UTC().Month() == month
	}
}

func isDayAfter(rule holidayRule) holidayRule {
	return func(date time.Time) bool {
		d := date.UTC()
		return rule(d.AddDate(0, 0, -1))
	}
}

func isFourthThursdayOfNovember(date time.Time) bool {
	d := date.UTC()
	if d.Month() != time.November {
		return false
	}
	year := d.Year()
	first := time.Date(year, time.November, 1, 0, 0, 0, 0, time.UTC)
	offset := (int(time.Thursday) - int(first.Weekday()) + 7) % 7
	fourthThursday := 1 + offset + 21
	return d.Day() == fourthThursday
}

func activeHolidayTaglines(now time.Time) []string {
	out := make([]string, 0, 2)
	for _, tagline := range holidayTaglines {
		rule := holidayRules[tagline]
		if rule != nil && rule(now) {
			out = append(out, tagline)
		}
	}
	return out
}

func allTaglines() []string {
	all := make([]string, 0, len(regularTaglines)+len(holidayTaglines))
	all = append(all, regularTaglines...)
	all = append(all, holidayTaglines...)
	return all
}

func pickTagline(now time.Time, getenv func(string) string, randFloat func() float64) string {
	if getenv != nil {
		if raw := getenv(envTaglineIndex); raw != "" {
			if idx, err := strconv.Atoi(raw); err == nil && idx >= 0 {
				all := allTaglines()
				if len(all) > 0 {
					return all[idx%len(all)]
				}
			}
		}
	}

	pool := regularTaglines
	if specials := activeHolidayTaglines(now); len(specials) > 0 {
		pool = specials
	}
	if len(pool) == 0 {
		return taglineDefault
	}

	randFn := randFloat
	if randFn == nil {
		randFn = taglineRand.Float64
	}
	idx := int(randFn() * float64(len(pool)))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(pool) {
		idx = len(pool) - 1
	}
	return pool[idx]
}
