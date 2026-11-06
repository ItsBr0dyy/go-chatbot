package commands

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"

	"github.com/itsbr0dyy/go-chatbot/utils"
	"github.com/itsbr0dyy/go-chatbot/utils/api/stv"
)

var emoteIDPattern = regexp.MustCompile(`^[A-Za-z0-9]{26}$`)

func init() {
	Register(&Command{
		Name:     "7tv",
		Aliases:  []string{"stv"},
		Category: "7TV",
		Usage:    "7tv add <emote|id> [bot] [#channel] [as:alias] | 7tv remove <emote> | 7tv alias <emote> <new alias>",
		Desc:     "Adds, removes or renames emotes in this channel's active 7TV set (7TV editors only).",
		Run:      stvCommand,
	})
}

func stvCommand(ctx *Context) {
	if ctx.Config.STVToken == "" {
		ctx.Reply("(7TV) 7TV isn't set up on this bot.")
		return
	}

	p := ctx.Config.Prefix
	usage := fmt.Sprintf("Usage: %s7tv add <emote|id> [bot] [#channel] [as:alias] | %s7tv remove <emote> | %s7tv alias <emote> <new alias>", p, p, p)
	if len(ctx.Args) == 0 {
		ctx.Reply(usage)
		return
	}

	c := stv.New(ctx.Config.STVToken)

	var msg string
	switch strings.ToLower(ctx.Args[0]) {
	case "add":
		msg = stvAdd(ctx, c)
	case "remove", "rm", "delete":
		msg = stvRemove(ctx, c)
	case "alias":
		msg = stvAlias(ctx, c)
	default:
		ctx.Reply(usage)
		return
	}
	ctx.Reply("(7TV) " + msg)
}

func stvError(err error, verb, emote string) string {
	log.Printf("7tv %s: %v", verb, err)

	if errors.Is(err, stv.ErrTokenInvalid) {
		return "My token has expired, please try again later or report it to the developer. NotLikeThis"
	}

	m := err.Error()
	switch {
	case strings.Contains(m, "INSUFFICIENT_PERMISSIONS"):
		return fmt.Sprintf("Insufficient permissions to %s emotes.", verb)
	case strings.Contains(m, "EMOTE_SET_FULL"):
		return "Emote set is full!"
	case strings.Contains(m, "BAD_REQUEST") && strings.Contains(m, "conflicting name"):
		return fmt.Sprintf("%s is already added to this set.", emote)
	case strings.Contains(m, "EMOTE_NOT_FOUND") || strings.Contains(strings.ToLower(m), "not found"):
		return fmt.Sprintf("Emote %q not found or is invalid.", emote)
	default:
		return fmt.Sprintf("Failed to %s emote %q.", verb, emote)
	}
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

var (
	botSTVMu sync.Mutex
	botSTVID string
)

func botSTV(ctx *Context, c *stv.Client) (string, error) {
	botSTVMu.Lock()
	defer botSTVMu.Unlock()

	if botSTVID != "" {
		return botSTVID, nil
	}
	u, err := utils.IVRUser(ctx.Config.Username)
	if err != nil {
		return "", err
	}
	id, err := c.UserIDFromTwitch(u.ID)
	if err != nil {
		return "", err
	}
	botSTVID = id
	return id, nil
}

func canEdit(ctx *Context, c *stv.Client, editors []string) (bool, error) {
	if strings.EqualFold(ctx.Msg.User, ctx.Msg.Channel) || ctx.MyStatus() >= StatusMod {
		return true, nil
	}
	id, err := c.UserIDFromTwitch(ctx.Msg.UserID())
	if err != nil {
		if errors.Is(err, stv.ErrUserNotFound) {
			return false, nil
		}
		return false, err
	}
	return contains(editors, id), nil
}

func stvOwner(ctx *Context, c *stv.Client, verb, emote string) (*stv.UserSet, string) {
	ownerID, err := c.UserIDFromTwitch(ctx.Msg.ChannelID())
	if err != nil {
		if errors.Is(err, stv.ErrUserNotFound) {
			return nil, "Could not find broadcaster's 7TV ID."
		}
		return nil, stvError(err, verb, emote)
	}

	owner, err := c.ActiveSet(ownerID)
	if err != nil {
		return nil, stvError(err, verb, emote)
	}

	allowed, err := canEdit(ctx, c, owner.Editors)
	if err != nil {
		return nil, stvError(err, verb, emote)
	}
	if !allowed {
		return nil, fmt.Sprintf("You must be a channel mod or 7TV editor to %s emotes.", verb)
	}
	return owner, ""
}

func stvAdd(ctx *Context, c *stv.Client) string {
	if len(ctx.Args) < 2 {
		return "Please provide an emote name or ID!"
	}
	target := ctx.Args[1]

	addToBot := false
	fromChannel := ""
	alias := ""
	for _, a := range ctx.Args[2:] {
		la := strings.ToLower(a)
		switch {
		case la == "bot":
			addToBot = true
		case strings.HasPrefix(la, "#") && len(la) > 1:
			fromChannel = strings.TrimPrefix(la, "#")
		case strings.HasPrefix(la, "as:") && len(a) > 3:
			alias = a[3:]
		}
	}
	if fromChannel != "" && !validLogName.MatchString(fromChannel) {
		return "That doesn't look like a valid channel name."
	}
	var aliasPtr *string
	if alias != "" {
		aliasPtr = &alias
	}

	owner, msg := stvOwner(ctx, c, "add", target)
	if msg != "" {
		return msg
	}

	botID, err := botSTV(ctx, c)
	if err != nil {
		return stvError(err, "add", target)
	}

	data := owner
	if addToBot {
		if data, err = c.ActiveSet(botID); err != nil {
			return stvError(err, "add", target)
		}
	}

	set := data.Set
	if set == nil {
		return "I cannot add emotes. No active emote set found."
	}
	if set.Emotes.TotalCount >= set.Capacity {
		return "Emote set is full! Please remove an emote first."
	}
	if !addToBot && !contains(owner.Editors, botID) {
		return "I cannot add emotes. Make sure I am a 7TV editor."
	}

	if alias != "" {
		for _, it := range set.Emotes.Items {
			if strings.EqualFold(it.Emote.DefaultName, alias) ||
				(it.Alias != nil && strings.EqualFold(*it.Alias, alias)) {
				return fmt.Sprintf("Alias %q conflicts with an existing emote. Either rename the alias or choose a different one.", alias)
			}
		}
	}

	type pick struct {
		id    string
		name  string
		alias *string
	}
	var p pick

	switch {
	case emoteIDPattern.MatchString(target):
		for _, it := range set.Emotes.Items {
			if it.Emote.ID == target {
				return "This emote is already in the set."
			}
		}
		p = pick{id: target, name: target, alias: aliasPtr}

	case fromChannel != "":
		src, err := utils.IVRUser(fromChannel)
		if err != nil {
			if errors.Is(err, utils.ErrTwitchUserNotFound) {
				return fmt.Sprintf("Could not find Twitch user %q.", fromChannel)
			}
			return stvError(err, "add", target)
		}
		srcSTV, err := c.UserIDFromTwitch(src.ID)
		if err != nil {
			if errors.Is(err, stv.ErrUserNotFound) {
				return fmt.Sprintf("Could not find 7TV profile for %q.", fromChannel)
			}
			return stvError(err, "add", target)
		}

		emotes, err := c.ChannelEmotes(srcSTV)
		if err != nil || len(emotes) == 0 {
			return fmt.Sprintf("Could not fetch emotes from #%s.", fromChannel)
		}

		var found *stv.ChannelEmote
		for i := range emotes {
			if strings.EqualFold(emotes[i].Name, target) {
				found = &emotes[i]
				break
			}
		}
		if found == nil {
			return fmt.Sprintf("%s not found in #%s's set.", target, fromChannel)
		}
		for _, it := range set.Emotes.Items {
			if it.Emote.ID == found.ID {
				return fmt.Sprintf("%s is already in the set.", target)
			}
		}

		a := found.Name
		if alias != "" {
			a = alias
		}
		p = pick{id: found.ID, name: found.Name, alias: &a}

	default:
		for _, it := range set.Emotes.Items {
			if strings.EqualFold(it.Emote.DefaultName, target) ||
				(it.Alias != nil && strings.EqualFold(*it.Alias, target)) {
				return fmt.Sprintf("%s is already in the set.", target)
			}
		}

		results, err := c.SearchEmotes(target)
		if err != nil {
			return stvError(err, "add", target)
		}
		if len(results) == 0 {
			return fmt.Sprintf("Emote %q not found.", target)
		}
		chosen := results[0]
		for _, r := range results {
			if strings.EqualFold(r.DefaultName, target) {
				chosen = r
				break
			}
		}
		p = pick{id: chosen.ID, name: chosen.DefaultName, alias: aliasPtr}
	}

	if err := c.AddEmote(set.ID, p.id, p.alias); err != nil {
		return stvError(err, "add", target)
	}

	name := p.name
	if p.alias != nil && *p.alias != "" {
		name = *p.alias
	}
	out := name + " has been added successfully"
	if fromChannel != "" {
		out += " from #" + fromChannel
	}
	if alias != "" {
		out += fmt.Sprintf(" as %q", alias)
	}
	return out + "!"
}

func stvRemove(ctx *Context, c *stv.Client) string {
	if len(ctx.Args) < 2 {
		return "Please provide an emote name!"
	}
	target := ctx.Args[1]

	owner, msg := stvOwner(ctx, c, "remove", target)
	if msg != "" {
		return msg
	}
	if owner.Set == nil {
		return "I cannot remove emotes. No active emote set found."
	}

	botID, err := botSTV(ctx, c)
	if err != nil {
		return stvError(err, "remove", target)
	}
	if !contains(owner.Editors, botID) {
		return "I cannot remove emotes. Make sure I am a 7TV editor."
	}

	var item *stv.SetEmote
	for i := range owner.Set.Emotes.Items {
		if strings.EqualFold(owner.Set.Emotes.Items[i].Name(), target) {
			item = &owner.Set.Emotes.Items[i]
			break
		}
	}
	if item == nil {
		return fmt.Sprintf("%s isn't in the set.", target)
	}

	if err := c.RemoveEmote(owner.Set.ID, item.Emote.ID, item.Alias); err != nil {
		return stvError(err, "remove", target)
	}
	return fmt.Sprintf("%s has been removed successfully!", item.Name())
}

func aliasError(err error, emote, alias string) string {
	// Expired token gets the standard message (and is logged) in stvError
	if errors.Is(err, stv.ErrTokenInvalid) {
		return stvError(err, "change", emote)
	}
	log.Printf("7tv alias: %v", err)

	m := err.Error()
	switch {
	case strings.Contains(m, "INSUFFICIENT_PERMISSIONS"):
		return "Insufficient permissions to modify emotes."
	case strings.Contains(m, "CONFLICTING_NAME") || strings.Contains(strings.ToLower(m), "conflicting"):
		return fmt.Sprintf("Alias %q conflicts with an existing emote.", alias)
	default:
		return fmt.Sprintf("Failed to update alias for %q.", emote)
	}
}

func stvAlias(ctx *Context, c *stv.Client) string {
	if len(ctx.Args) < 3 {
		return fmt.Sprintf("Usage: %s7tv alias <emote name> <new alias>", ctx.Config.Prefix)
	}
	emoteName, newAlias := ctx.Args[1], ctx.Args[2]

	owner, msg := stvOwner(ctx, c, "change", emoteName)
	if msg != "" {
		return msg
	}
	if owner.Set == nil {
		return "No active emote set found."
	}

	botID, err := botSTV(ctx, c)
	if err != nil {
		return aliasError(err, emoteName, newAlias)
	}
	if !contains(owner.Editors, botID) {
		return "I cannot modify emotes. Make sure I am a 7TV editor."
	}

	var item *stv.SetEmote
	for i := range owner.Set.Emotes.Items {
		it := &owner.Set.Emotes.Items[i]
		if strings.EqualFold(it.Emote.DefaultName, emoteName) ||
			(it.Alias != nil && strings.EqualFold(*it.Alias, emoteName)) {
			item = it
			break
		}
	}
	if item == nil {
		return fmt.Sprintf("Emote %q not found in the emote set.", emoteName)
	}

	for _, it := range owner.Set.Emotes.Items {
		if it.ID == item.ID {
			continue
		}
		if strings.EqualFold(it.Emote.DefaultName, newAlias) ||
			(it.Alias != nil && strings.EqualFold(*it.Alias, newAlias)) {
			return fmt.Sprintf("Alias %s conflicts with an existing emote.", newAlias)
		}
	}

	if err := c.UpdateEmoteAlias(owner.Set.ID, item.Emote.ID, newAlias); err != nil {
		return aliasError(err, emoteName, newAlias)
	}
	return fmt.Sprintf("Successfully changed %s to %s", item.Name(), newAlias)
}
