package commands

import (
	"fmt"
	"runtime"
	"strconv"
	"time"

	"github.com/itsbr0dyy/go-chatbot/utils"
)

func init() {
	Register(&Command{
		Name:     "ping",
		Category: "General",
		Usage:    "ping",
		Desc:     "Shows message delay, uptime and memory use.",
		Run:      ping,
	})
}

func ping(ctx *Context) {
	msgDelay := "n/a"
	if ts, err := strconv.ParseInt(ctx.Msg.Tags["tmi-sent-ts"], 10, 64); err == nil {
		d := time.Since(time.UnixMilli(ts)).Milliseconds()
		if d < 0 {
			d = 0
		}
		msgDelay = fmt.Sprintf("%dms", d)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	ctx.Reply(fmt.Sprintf(
		"Pong! Msg delay: %s | Uptime: %s | Mem: %.1f MB",
		msgDelay,
		utils.FormatDuration(utils.Uptime()),
		float64(m.Alloc)/1024/1024,
	))
}
