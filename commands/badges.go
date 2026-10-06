package commands

import (
	"fmt"
	"strings"
	"time"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
)

var excludedBadgeIDs = map[string]bool{
	"bloom-badge":   true,
	"blossom-badge": true,
	"turbo":         true,
}

var excludedBadgePrefixes = []string{"ewc-2026-co-streamer-"}

func badgeExcluded(id string) bool {
	if excludedBadgeIDs[id] {
		return true
	}
	for _, p := range excludedBadgePrefixes {
		if strings.HasPrefix(id, p) {
			return true
		}
	}
	return false
}

func init() {
	Register(&Command{
		Name:     "badges",
		Category: "Twitch info",
		Usage:    "badges [username]",
		Desc:     "Shows which active event badges a user is missing and when the next one drops.",
		Run:      badges,
	})
}

func badges(ctx *Context) {
	target := ctx.Msg.User
	name := ctx.Msg.DisplayName()
	if len(ctx.Args) > 0 {
		target = cleanName(ctx.Args[0])
		name = target
		if target == ctx.Msg.User {
			name = ctx.Msg.DisplayName()
		}
	}
	if !validLogName.MatchString(target) {
		ctx.Reply(fmt.Sprintf("Usage: %sbadges [username]", ctx.Config.Prefix))
		return
	}

	all, missing, err := utils.CatQueryBadges(target)
	if err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't get badge info: %v", err))
		return
	}

	now := time.Now()

	activeByID := map[string]utils.EventBadge{}
	activeCount := 0
	var next *utils.EventBadge
	var nextStart time.Time
	for i := range all {
		b := all[i]
		if badgeExcluded(b.ID) {
			continue
		}
		start, okStart := b.Start()
		end, okEnd := b.End()

		if okStart && okEnd && !start.After(now) && !end.Before(now) {
			activeByID[b.ID] = b
			activeCount++
		} else if okStart && start.After(now) && (next == nil || start.Before(nextStart)) {
			next = &all[i]
			nextStart = start
		}
	}

	var missingNames []string
	freeCount := 0
	for _, m := range missing {
		b, ok := activeByID[m.ID]
		if !ok {
			continue
		}
		label := b.Name
		if label == "" {
			label = b.ID
		}
		if m.Free {
			freeCount++
			label += " (free)"
		}
		missingNames = append(missingNames, label)
	}

	var parts []string

	switch {
	case activeCount == 0:
		parts = append(parts, "There are no active event badges right now")
	case len(missingNames) == 0:
		parts = append(parts, fmt.Sprintf("%s has every active badge (%d/%d)", name, activeCount, activeCount))
	default:
		parts = append(parts, fmt.Sprintf("%s is missing %d/%d active badges (%d free)",
			name, len(missingNames), activeCount, freeCount))

		const maxLen = 220
		var shown []string
		length := 0
		for _, n := range missingNames {
			if length+len(n)+2 > maxLen {
				break
			}
			shown = append(shown, n)
			length += len(n) + 2
		}
		list := strings.Join(shown, ", ")
		if rest := len(missingNames) - len(shown); rest > 0 {
			list += fmt.Sprintf(" and %d more", rest)
		}
		parts = append(parts, "Missing: "+list)
	}

	if next != nil {
		nextName := next.Name
		if nextName == "" {
			nextName = next.ID
		}
		parts = append(parts, fmt.Sprintf("Next: %q in %s", nextName, utils.FormatLong(nextStart.Sub(now))))
	}

	link := "https://twitchbadge.com/" + target
	if short, err := utils.Shorten(ctx.Config.ShortAPI, link); err == nil {
		link = short
	}
	parts = append(parts, link)

	ctx.Reply(strings.Join(parts, " • "))
}
