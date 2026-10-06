package commands

import (
	"errors"
	"fmt"
	"strings"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
)

func init() {
	Register(&Command{
		Name:     "song",
		Category: "Music",
		Usage:    "song [username]",
		Desc:     "Shows what you or another linked user is listening to.",
		Run:      song,
	})
}

func song(ctx *Context) {
	if ctx.Config.LastFMKey == "" {
		ctx.Reply("Last.fm isn't set up on this bot.")
		return
	}

	target := ctx.Msg.User
	name := ctx.Msg.DisplayName()
	self := true
	if len(ctx.Args) > 0 {
		target = strings.ToLower(strings.TrimPrefix(ctx.Args[0], "@"))
		name = target
		self = strings.EqualFold(target, ctx.Msg.User)
		if self {
			name = ctx.Msg.DisplayName()
		}
	}

	lfm, ok := ctx.Links.Get("lastfm", target)
	if !ok {
		if self {
			ctx.Reply(fmt.Sprintf("You haven't linked a Last.fm account. Use %slink lastfm <username>", ctx.Config.Prefix))
		} else {
			ctx.Reply(fmt.Sprintf("%s hasn't linked a Last.fm account.", target))
		}
		return
	}

	track, err := utils.LastFMRecent(ctx.Config.LastFMKey, lfm)
	if err != nil {
		if errors.Is(err, utils.ErrNoTracks) {
			ctx.Reply("No scrobbles found for that account.")
		} else {
			ctx.Reply(fmt.Sprintf("Couldn't reach Last.fm: %v", err))
		}
		return
	}

	verb := "last listened to"
	if track.NowPlaying {
		verb = "is currently listening to"
	}

	parts := []string{fmt.Sprintf("%s %s \"%s\" by %s", name, verb, track.Name, track.Artist)}

	if track.Album != "" {
		parts = append(parts, fmt.Sprintf("from the album \"%s\"", track.Album))
	}

	if plays, err := utils.LastFMPlayCount(ctx.Config.LastFMKey, lfm, track.Artist, track.Name); err == nil && plays > 0 {
		parts = append(parts, fmt.Sprintf("plays #%d %s", plays, ctx.Config.SongEmote))
	}

	if track.URL != "" {
		link, err := utils.Shorten(ctx.Config.ShortAPI, track.URL)
		if err != nil {
			link = track.URL
		}
		parts = append(parts, link)
	}

	ctx.Reply(strings.Join(parts, " "))
}
