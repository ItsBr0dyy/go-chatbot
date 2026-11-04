package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Span struct {
	Years, Months, Days int
	D                   time.Duration
}

func (s *Span) add(o Span) {
	s.Years += o.Years
	s.Months += o.Months
	s.Days += o.Days
	s.D += o.D
}

func (s Span) Apply(t time.Time) time.Time {
	return t.AddDate(s.Years, s.Months, s.Days).Add(s.D)
}

var spanPart = regexp.MustCompile(`^(\d+)([a-z]+)`)

const maxSpanPart = 1_000_000

/* supports 1d; 12h; 3months; 1h30m; */
func parseSpanToken(tok string) (Span, bool) {
	var s Span
	tok = strings.ToLower(strings.TrimSpace(tok))
	if tok == "" {
		return s, false
	}

	for tok != "" {
		m := spanPart.FindStringSubmatch(tok)
		if m == nil {
			return Span{}, false
		}
		n, err := strconv.Atoi(m[1])
		if err != nil || n > maxSpanPart {
			return Span{}, false
		}

		switch m[2] {
		case "s", "sec", "secs", "second", "seconds":
			s.D += time.Duration(n) * time.Second
		case "m", "min", "mins", "minute", "minutes":
			s.D += time.Duration(n) * time.Minute
		case "h", "hr", "hrs", "hour", "hours":
			s.D += time.Duration(n) * time.Hour
		case "d", "day", "days":
			s.Days += n
		case "w", "wk", "wks", "week", "weeks":
			s.Days += n * 7
		case "mo", "mos", "month", "months":
			s.Months += n
		case "y", "yr", "yrs", "year", "years":
			s.Years += n
		default:
			return Span{}, false
		}
		tok = tok[len(m[0]):]
	}
	return s, true
}

func ParseSpan(args []string) (Span, int) {
	var total Span
	used := 0
	for _, a := range args {
		s, ok := parseSpanToken(a)
		if !ok {
			break
		}
		total.add(s)
		used++
	}
	return total, used
}

func FormatLong(d time.Duration) string {
	if d < time.Minute {
		return "less than a minute"
	}

	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60

	var parts []string
	add := func(n int, unit string) {
		if n == 0 {
			return
		}
		if n != 1 {
			unit += "s"
		}
		parts = append(parts, fmt.Sprintf("%d %s", n, unit))
	}
	add(days, "day")
	add(hours, "hour")
	add(mins, "minute")

	switch len(parts) {
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}

func FormatAgo(since time.Time) string {
	d := time.Since(since)
	if d < 0 {
		d = 0
	}

	days := int(d.Hours() / 24)
	units := []struct {
		n    int
		name string
	}{
		{days / 365, "y"},
		{(days % 365) / 30, "mo"},
		{(days % 365) % 30, "d"},
		{int(d.Hours()) % 24, "h"},
		{int(d.Minutes()) % 60, "m"},
	}

	for i, u := range units {
		if u.n == 0 {
			continue
		}
		out := fmt.Sprintf("%d%s", u.n, u.name)
		if i+1 < len(units) && units[i+1].n > 0 {
			out += fmt.Sprintf(" %d%s", units[i+1].n, units[i+1].name)
		}
		return out
	}
	return "<1m"
}
