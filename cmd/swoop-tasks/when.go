// Date words: what turns "buy milk tomorrow" into a task called "buy
// milk" due tomorrow. Only the end of the text is read, and only a few
// words, so a task that merely mentions a day ("call Friday's venue")
// keeps it. Every function here takes the time as an argument; nothing
// reads the clock, so the tests can pick the day.
package main

import (
	"fmt"
	"strings"
	"time"
)

// iso is the date format in the file: 2026-10-01.
const iso = "2006-01-02"

// weekdays maps the names a person types to a day of the week: the whole
// word and the usual short forms.
var weekdays = map[string]time.Weekday{
	"sunday": time.Sunday, "sun": time.Sunday,
	"monday": time.Monday, "mon": time.Monday,
	"tuesday": time.Tuesday, "tue": time.Tuesday, "tues": time.Tuesday,
	"wednesday": time.Wednesday, "wed": time.Wednesday,
	"thursday": time.Thursday, "thu": time.Thursday, "thur": time.Thursday, "thurs": time.Thursday,
	"friday": time.Friday, "fri": time.Friday,
	"saturday": time.Saturday, "sat": time.Saturday,
}

// connectors are the small words that can sit before a date and go with
// it: "report by friday", "dentist on tuesday", "rent due 2026-10-01".
// "next" is taken as said: "next friday" is the coming Friday, the same
// as "friday", which is what most people mean by it on a Monday.
var connectors = map[string]bool{"by": true, "on": true, "due": true, "next": true, "this": true}

// Split takes the date words off the end of text. It returns the text
// without them and the due day, or the text as it was and "" when it
// does not end in one. The words: "today" (or "tonight"), "tomorrow", a
// weekday, which means the next one to come, never today, and a date as
// 2026-10-01. Text that is only a date word is a task called that word:
// something has to be left to do.
func Split(text string, now time.Time) (string, string) {
	words := strings.Fields(text)
	if len(words) < 2 {
		return strings.TrimSpace(text), ""
	}
	due, ok := word(words[len(words)-1], now)
	if !ok {
		return strings.Join(words, " "), ""
	}
	words = words[:len(words)-1]
	for len(words) > 1 && connectors[strings.ToLower(words[len(words)-1])] {
		words = words[:len(words)-1]
	}
	return strings.Join(words, " "), due.Format(iso)
}

// word reads one date word, relative to now's day.
func word(w string, now time.Time) (time.Time, bool) {
	today := day(now)
	lw := strings.ToLower(strings.TrimRight(w, ".,!"))
	switch lw {
	case "today", "tonight":
		return today, true
	case "tomorrow":
		return today.AddDate(0, 0, 1), true
	}
	if wd, ok := weekdays[lw]; ok {
		ahead := (int(wd) - int(today.Weekday()) + 7) % 7
		if ahead == 0 {
			ahead = 7
		}
		return today.AddDate(0, 0, ahead), true
	}
	if t, err := time.ParseInLocation(iso, lw, now.Location()); err == nil {
		return t, true
	}
	return time.Time{}, false
}

// day is the midnight that starts t's day, in t's zone.
func day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// daysUntil is how many days from now's day to the due day: 0 today,
// 1 tomorrow, negative when it has passed. Counted on calendar dates, so
// a change to or from summer time does not make a day 23 hours short.
func daysUntil(due string, now time.Time) (int, bool) {
	t, err := time.ParseInLocation(iso, due, now.Location())
	if err != nil {
		return 0, false
	}
	a, b := day(now), t
	n := 0
	for a.Before(b) {
		a = a.AddDate(0, 0, 1)
		n++
	}
	for b.Before(a) {
		b = b.AddDate(0, 0, 1)
		n--
	}
	return n, true
}

// Label says when a due day is, in words for a subtitle: "today",
// "tomorrow", "Friday" within the week, "Fri 9 Oct" beyond it, the year
// too when it is not this one, and "overdue" in front of a day gone by.
func Label(due string, now time.Time) string {
	n, ok := daysUntil(due, now)
	if !ok {
		return due
	}
	t, _ := time.ParseInLocation(iso, due, now.Location())
	date := t.Format("Mon 2 Jan")
	if t.Year() != now.Year() {
		date = t.Format("Mon 2 Jan 2006")
	}
	switch {
	case n < 0:
		return "overdue, " + date
	case n == 0:
		return "today"
	case n == 1:
		return "tomorrow, " + date
	case n < 7:
		return t.Weekday().String()
	}
	return date
}

// describe is the due day as the Add row and the preview say it.
func describe(due string, now time.Time) string {
	if due == "" {
		return "no due date"
	}
	return fmt.Sprintf("due %s", Label(due, now))
}
