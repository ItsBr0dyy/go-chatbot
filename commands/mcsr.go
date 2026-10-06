package commands

import (
	"errors"
	"fmt"
	"strings"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
)

type mcsrView func(ctx *Context, u *utils.MCSRUser, st utils.MCSRStats, scope string) string

func init() {
	Register(&Command{
		Name: "elo", Category: "MCSR", Usage: "elo [minecraft username]",
		Desc: "Shows a player's MCSR Ranked elo and rank.",
		Run:  mcsrCommand(viewElo),
	})
	Register(&Command{
		Name: "winrate", Category: "MCSR", Usage: "winrate [minecraft username] [total]",
		Desc: "Shows a player's MCSR Ranked winrate (this season, or all time with 'total').",
		Run:  mcsrCommand(viewWinrate),
	})
	Register(&Command{
		Name: "record", Category: "MCSR", Usage: "record [minecraft username] [total]",
		Desc: "Shows wins, losses, forfeits and best win streak.",
		Run:  mcsrCommand(viewRecord),
	})
	Register(&Command{
		Name: "average", Category: "MCSR", Usage: "average [minecraft username] [total]",
		Desc: "Shows a player's average completion time and PB.",
		Run:  mcsrCommand(viewAverage),
	})
}

func mcsrCommand(view mcsrView) func(ctx *Context) {
	return func(ctx *Context) {
		total := false
		var args []string
		for _, a := range ctx.Args {
			if strings.EqualFold(a, "total") || strings.EqualFold(a, "all") {
				total = true
				continue
			}
			args = append(args, a)
		}

		var name string
		if len(args) > 0 {
			name = strings.TrimPrefix(args[0], "@")
			if !validMCName.MatchString(name) {
				ctx.Reply("That doesn't look like a valid Minecraft username.")
				return
			}
		} else {
			linked, ok := ctx.Links.Get("mc", ctx.Msg.User)
			if !ok {
				ctx.Reply(fmt.Sprintf("You haven't linked a Minecraft account. Use %slink mc <username>", ctx.Config.Prefix))
				return
			}
			name = linked
		}

		u, err := utils.MCSRLookup(name)
		if err != nil {
			if errors.Is(err, utils.ErrMCSRNotFound) {
				ctx.Reply(fmt.Sprintf("No MCSR Ranked profile found for %s.", name))
			} else {
				ctx.Reply(fmt.Sprintf("Couldn't reach MCSR Ranked: %v", err))
			}
			return
		}

		st, scope := u.Statistics.Season, "this season"
		if total {
			st, scope = u.Statistics.Total, "all time"
		}
		ctx.Reply(view(ctx, u, st, scope))
	}
}

func viewElo(ctx *Context, u *utils.MCSRUser, _ utils.MCSRStats, _ string) string {
	if u.EloRate == nil {
		return fmt.Sprintf("%s hasn't finished their placement matches yet.", u.Nickname)
	}

	st := u.Statistics.Season
	elo := *u.EloRate

	head := fmt.Sprintf("%s Stats: Elo %d", u.Nickname, elo)
	if sr := u.SeasonResult; sr != nil && sr.Highest != nil {
		peak := *sr.Highest
		if elo > peak {
			peak = elo
		}
		head += fmt.Sprintf(" (Peak %d)", peak)
	}
	parts := []string{head}

	tier := utils.MCSRTier(elo)
	if u.EloRank != nil {
		tier += fmt.Sprintf(" (#%d)", *u.EloRank)
	}
	parts = append(parts, tier)

	wins, losses := st.Wins.R(), st.Loses.R()
	if wins+losses > 0 {
		rate := float64(wins) / float64(wins+losses) * 100
		parts = append(parts, fmt.Sprintf("W/L: %d/%d (%.1f%%)", wins, losses, rate))
	}

	played := st.PlayedMatches.R()
	if played > 0 {
		parts = append(parts, fmt.Sprintf("Played %d matches", played))
	}

	if pb := st.BestTime.Ranked; pb != nil {
		t := "Fastest time " + utils.FormatMinSec(*pb)
		if c := st.Completions.R(); c > 0 {
			t += fmt.Sprintf(" (avg %s)", utils.FormatMinSec(st.CompletionTime.R()/c))
		}
		parts = append(parts, t)
	}

	if played > 0 {
		ff := float64(st.Forfeits.R()) / float64(played) * 100
		parts = append(parts, fmt.Sprintf("FF rate %.2f%%", ff))
	}

	return strings.Join(parts, " • ")
}

func viewWinrate(ctx *Context, u *utils.MCSRUser, st utils.MCSRStats, scope string) string {
	wins, losses := st.Wins.R(), st.Loses.R()
	if wins+losses == 0 {
		return fmt.Sprintf("%s hasn't played any ranked matches %s.", u.Nickname, scope)
	}
	rate := float64(wins) / float64(wins+losses) * 100
	return fmt.Sprintf("%s has a %.1f%% winrate %s (%dW - %dL)", u.Nickname, rate, scope, wins, losses)
}

func viewRecord(ctx *Context, u *utils.MCSRUser, st utils.MCSRStats, scope string) string {
	played := st.PlayedMatches.R()
	if played == 0 {
		return fmt.Sprintf("%s hasn't played any ranked matches %s.", u.Nickname, scope)
	}

	parts := []string{
		fmt.Sprintf("%dW - %dL", st.Wins.R(), st.Loses.R()),
		fmt.Sprintf("%d played", played),
	}
	forfeits := st.Forfeits.R()
	parts = append(parts, fmt.Sprintf("%d forfeits (%.1f%%)", forfeits, float64(forfeits)/float64(played)*100))
	parts = append(parts, fmt.Sprintf("best win streak %d", st.HighestWinStreak.R()))

	return fmt.Sprintf("%s's record %s: %s", u.Nickname, scope, strings.Join(parts, " • "))
}

func viewAverage(ctx *Context, u *utils.MCSRUser, st utils.MCSRStats, scope string) string {
	completions := st.Completions.R()
	if completions == 0 {
		return fmt.Sprintf("%s has no completed ranked runs %s.", u.Nickname, scope)
	}

	avg := st.CompletionTime.R() / completions
	msg := fmt.Sprintf("%s's average time %s is %s over %d completions",
		u.Nickname, scope, utils.FormatMS(avg), completions)

	if pb := st.BestTime.Ranked; pb != nil {
		msg += " • PB " + utils.FormatMS(*pb)
	}
	return msg
}
