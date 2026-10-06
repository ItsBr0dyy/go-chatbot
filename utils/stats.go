package utils

import (
	"fmt"
	"sync/atomic"
	"time"
)

var (
	StartTime   = time.Now()
	commandsRun atomic.Int64
)

func CountCommand()         { commandsRun.Add(1) }
func CommandsRun() int64    { return commandsRun.Load() }
func Uptime() time.Duration { return time.Since(StartTime).Round(time.Second) }

func FormatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60

	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	case hours > 0:
		return fmt.Sprintf("%dh %dm %ds", hours, mins, secs)
	case mins > 0:
		return fmt.Sprintf("%dm %ds", mins, secs)
	default:
		return fmt.Sprintf("%ds", secs)
	}
}
