package commands

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/itsbr0dyy/go-chatbot/utils"
	"github.com/itsbr0dyy/go-chatbot/utils/api/stv"
)

const stvTokenExpired = "My token has expired, please try again later or report it to the developer."

var digitsOnly = regexp.MustCompile(`^\d+$`)

var stvBadgeTiers = []struct {
	name string
	days int
}{
	{"1 Month", 30},
	{"3 Months", 90},
	{"6 Months", 180},
	{"9 Months", 270},
	{"1 Year", 365},
	{"2 Years", 730},
	{"3 Years", 1095},
	{"4 Years", 1460},
	{"5 Years", 1825},
	{"6 Years", 2190},
	{"7 Years", 2555},
}

func init() {
	Register(&Command{
		Name:     "7tvuser",
		Aliases:  []string{"7tvu"},
		Category: "7TV",
		Usage:    "7tvuser [username]",
		Desc:     "Shows a user's 7TV profile, roles, paint, badge and editors.",
		Run:      stvUserCmd,
	})
	Register(&Command{
		Name:     "7tvsubage",
		Aliases:  []string{"7tvsa"},
		Category: "7TV",
		Usage:    "7tvsubage [username]",
		Desc:     "Shows a user's 7TV subscription: months, renewal, gifter and badge progress.",
		Run:      stvSubageCmd,
	})
}

func monthsText(n int) string {
	if n == 1 {
		return "1 month"
	}
	return fmt.Sprintf("%d months", n)
}

func stvUserCmd(ctx *Context) {
	target := ctx.Msg.User
	if len(ctx.Args) > 0 {
		target = cleanName(ctx.Args[0])
	}
	if !validLogName.MatchString(target) {
		ctx.Reply(fmt.Sprintf("Usage: %s7tvuser [username]", ctx.Config.Prefix))
		return
	}

	u, err := utils.IVRUser(target)
	if err != nil {
		if errors.Is(err, utils.ErrTwitchUserNotFound) {
			ctx.Reply(fmt.Sprintf("User %s not found.", target))
		} else {
			ctx.Reply(fmt.Sprintf("Could not fetch 7TV info for %s.", target))
		}
		return
	}
	name := u.DisplayName
	if name == "" {
		name = u.Login
	}

	p, err := stv.New("").UserProfile(u.ID)
	if err != nil {
		if errors.Is(err, stv.ErrUserNotFound) {
			ctx.Reply(fmt.Sprintf("(7TV) %s has no 7TV connection.", name))
		} else {
			log.Printf("7tvuser: %v", err)
			ctx.Reply(fmt.Sprintf("Could not fetch 7TV info for %s.", target))
		}
		return
	}

	short := func(link string) string {
		if s, err := utils.Shorten(ctx.Config.ShortAPI, link); err == nil {
			return s
		}
		return link
	}

	roles := "None"
	if len(p.Roles) > 0 {
		roles = strings.Join(p.Roles, ", ")
	}
	paint, badge := "None", "None"
	if p.PaintName != "" {
		paint = p.PaintName
	}
	if p.BadgeName != "" {
		badge = p.BadgeName
	}

	parts := []string{
		name,
		"7TV profile: " + short("https://7tv.app/users/"+p.ID),
		"roles: " + roles,
		"paint: " + paint,
		"badge: " + badge,
		fmt.Sprintf("editors: %d", p.Editors),
	}
	if !p.CreatedAt.IsZero() {
		parts = append(parts, "created: "+utils.FormatAgo(p.CreatedAt)+" ago")
	}
	if p.PaintID != "" {
		parts = append(parts, "try on paint: "+short("https://7database.com/paint/"+p.PaintID))
	}
	if p.BadgeID != "" {
		parts = append(parts, "view Badge: "+short("https://7database.com/badge/"+p.BadgeID))
	}

	ctx.Reply("(7TV) " + strings.Join(parts, " • "))
}

func stvSubageCmd(ctx *Context) {
	if ctx.Config.STVToken == "" {
		ctx.Reply("(7TV) 7TV isn't set up on this bot.")
		return
	}

	platformID := ctx.Msg.UserID()
	name := ctx.Msg.DisplayName()

	if len(ctx.Args) > 0 {
		arg := strings.ToLower(strings.TrimPrefix(ctx.Args[0], "@"))
		if digitsOnly.MatchString(arg) {
			platformID, name = arg, arg
		} else {
			if !validLogName.MatchString(arg) {
				ctx.Reply(fmt.Sprintf("Usage: %s7tvsubage [username]", ctx.Config.Prefix))
				return
			}
			u, err := utils.IVRUser(arg)
			if err != nil {
				if errors.Is(err, utils.ErrTwitchUserNotFound) {
					ctx.Reply(fmt.Sprintf("(7TV) Could not find user %q.", arg))
				} else {
					ctx.Reply("(7TV) Failed to fetch subscription information.")
				}
				return
			}
			platformID, name = u.ID, u.Login
		}
	}

	c := stv.New(ctx.Config.STVToken)
	sub, err := c.UserSubscription(platformID)
	if err != nil {
		switch {
		case errors.Is(err, stv.ErrTokenInvalid):
			log.Printf("7tvsubage: %v", err)
			ctx.Reply("(7TV) " + stvTokenExpired)
		case errors.Is(err, stv.ErrUserNotFound):
			ctx.Reply(fmt.Sprintf("(7TV) %s doesn't have a 7TV account.", name))
		default:
			log.Printf("7tvsubage: %v", err)
			ctx.Reply("(7TV) Failed to fetch subscription information.")
		}
		return
	}

	if sub == nil || !sub.Active {
		ctx.Reply(fmt.Sprintf("(7TV) %s does not have an active 7TV subscription.", name))
		return
	}

	left := time.Until(sub.End)
	if left < 0 {
		left = 0
	}
	days := int(left.Hours()) / 24
	hours := int(left.Hours()) % 24

	status := "ending"
	if sub.AutoRenew {
		status = "renewing"
	}

	parts := []string{
		fmt.Sprintf("%s has been subscribed for %s", name, monthsText(sub.TotalDays/30)),
		fmt.Sprintf("%s in %dd %dh", status, days, hours),
	}

	if sub.GiftedBy != "" {
		if gifter, err := c.DisplayNameV3(sub.GiftedBy); err == nil && gifter != "" {
			parts = append(parts, "gifted by "+gifter)
		}
	}

	badgeText := "Max badge tier reached!"
	for _, tier := range stvBadgeTiers {
		if sub.TotalDays < tier.days {
			badgeText = fmt.Sprintf("%d%% progress towards %q badge (%d days left)",
				sub.TotalDays*100/tier.days, tier.name, tier.days-sub.TotalDays)
			break
		}
	}
	parts = append(parts, badgeText)

	ctx.Reply("(7TV) " + strings.Join(parts, " • "))
}
