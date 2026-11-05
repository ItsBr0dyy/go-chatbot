package commands

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/itsbr0dyy/go-chatbot/config"
	"github.com/itsbr0dyy/go-chatbot/utils"
)

const (
	catCooldown     = 30 * time.Minute
	cdrCost         = 10
	catMissChance   = 25
	leaderboardSize = 10
)

var catWeights = []int{
	200, 150, 120, 95, 75,
	60, 48, 38, 30, 24,
	19, 15, 12, 9, 7,
	4, 3, 2, 1, 1,
}

var (
	catMissMessages = []string{
		"You rattled the treat bag but no cat showed up.",
		"A cat watched you from a fence, then walked away.",
		"You chased a cat and it vanished into the bushes.",
		"The cat you were after turned out to be a very convincing plant.",
		"A cat meowed at you from across the street and ignored the treat.",
		"You were one second late and the cat slipped out the window.",
		"A cat looked at you, yawned and went back to sleep somewhere you can't reach.",
	}
	// exactly 1 cat
	catOneMessages = []string{
		"You offered a bowl of tuna and a stray cat decided to stay! +{n}",
		"A cat jumped into your lap out of nowhere. +{n}",
		"You shook the treat bag and a cat came running. +{n}",
		"A sleepy cat followed you home. +{n}",
		"You found a cat napping in a cardboard box. +{n}",
		"A cat knocked your cup off the table, then adopted you. +{n}",
		"You made a kissy noise and a cat answered. +{n}",
		"A cat stared at you for ten minutes, then climbed on your shoulder. +{n}",
	}
	// 2 to 5 cats
	catFewMessages = []string{
		"A cat showed up with some friends! +{n}",
		"You opened a can and a few cats appeared from nowhere. +{n}",
		"A mama cat led her kittens right to your door. +{n}",
		"A small gang of cats followed you home. +{n}",
	}
	// 6 to 15 cats
	catManyMessages = []string{
		"A whole cat parade marched past and some of them stayed. +{n}",
		"You found a box of kittens on the porch. +{n}",
		"You walked into a cat cafe that was giving cats away. +{n}",
		"A truck full of cats broke down outside your house. +{n}",
	}
	// 16 to 20 cats (rare)
	catLegendaryMessages = []string{
		"A legendary cat king arrived with his entire court. +{n}",
		"You opened the door and it was raining cats. +{n}",
		"Every cat in the neighborhood showed up at once. +{n}",
	}
)

func init() {
	Register(&Command{
		Name:     "cat",
		Category: "Cats",
		Usage:    "cat",
		Desc:     "Try to catch some cats, up to 20 at a time (big hauls are rare). 30 minute cooldown.",
		Run:      catCmd,
	})
	Register(&Command{
		Name:     "cdr",
		Category: "Cats",
		Usage:    "cdr",
		Desc:     fmt.Sprintf("Resets your cat cooldown for %d cats.", cdrCost),
		Run:      cdrCmd,
	})
	Register(&Command{
		Name:     "cats",
		Category: "Cats",
		Usage:    "cats",
		Desc:     "Shows how many cats you have.",
		Run:      catsCmd,
	})
	Register(&Command{
		Name:     "give",
		Category: "Cats",
		Usage:    "give <user> <amount>",
		Desc:     "Gives some of your cats to another user.",
		Run:      giveCmd,
	})
	Register(&Command{
		Name:     "leaderboard",
		Category: "Cats",
		Usage:    "leaderboard",
		Desc:     "Shows the top cat owners.",
		Run:      leaderboardCmd,
	})
	Register(&Command{
		Name:     "rank",
		Category: "Cats",
		Usage:    "rank [user]",
		Desc:     "Shows your (or another user's) cat rank and amount.",
		Run:      rankCmd,
	})
}

func catsText(n int64) string {
	if n == 1 {
		return "1 cat"
	}
	return fmt.Sprintf("%s cats", utils.FormatInt(int(n)))
}

func pick(list []string) string {
	return list[rand.IntN(len(list))]
}

func rollAmount() int64 {
	total := 0
	for _, w := range catWeights {
		total += w
	}

	r := rand.IntN(total)
	for i, w := range catWeights {
		if r < w {
			return int64(i + 1)
		}
		r -= w
	}
	return 1
}

func catMessage(n int64) string {
	switch {
	case n == 1:
		return pick(catOneMessages)
	case n <= 5:
		return pick(catFewMessages)
	case n <= 15:
		return pick(catManyMessages)
	default:
		return pick(catLegendaryMessages)
	}
}

func rollCat() (int64, string) {
	if rand.IntN(100) < catMissChance {
		return 0, pick(catMissMessages)
	}
	n := rollAmount()
	return n, catMessage(n)
}

func catErr(ctx *Context, err error) {
	ctx.Reply(fmt.Sprintf("Something went wrong with the cats: %v", err))
}

func catTarget(ctx *Context, arg string) (string, string, error) {
	login := cleanName(arg)
	if login == ctx.Msg.User {
		return ctx.Msg.UserID(), ctx.Msg.User, nil
	}

	acc, err := ctx.Cats.FindByLogin(login)
	if err != nil {
		return "", "", err
	}
	if acc != nil {
		return acc.UserID, acc.Login, nil
	}

	u, err := utils.IVRUser(login)
	if err != nil {
		return "", "", err
	}
	return u.ID, strings.ToLower(u.Login), nil
}

func cooldownLeft(acc *config.CatAccount) time.Duration {
	if acc == nil || acc.LastCatch == nil {
		return 0
	}
	return catCooldown - time.Since(*acc.LastCatch)
}

func catCmd(ctx *Context) {
	gain, msg := rollCat()

	balance, ok, err := ctx.Cats.Catch(ctx.Msg.UserID(), ctx.Msg.User, gain, catCooldown)
	if err != nil {
		catErr(ctx, err)
		return
	}

	if !ok {
		acc, err := ctx.Cats.Get(ctx.Msg.UserID())
		if err != nil {
			catErr(ctx, err)
			return
		}
		ctx.Reply(fmt.Sprintf("You can catch another cat in %s. (%scdr resets it for %s)",
			utils.FormatLong(cooldownLeft(acc)), ctx.Config.Prefix, catsText(cdrCost)))
		return
	}

	msg = strings.ReplaceAll(msg, "{n}", catsText(gain))
	if gain == 0 {
		ctx.Reply(fmt.Sprintf("%s You still have %s.", msg, catsText(balance)))
	} else {
		ctx.Reply(fmt.Sprintf("%s You have %s now.", msg, catsText(balance)))
	}
}

func cdrCmd(ctx *Context) {
	acc, err := ctx.Cats.Get(ctx.Msg.UserID())
	if err != nil {
		catErr(ctx, err)
		return
	}

	if cooldownLeft(acc) <= 0 {
		ctx.Reply("You don't have a cooldown to reset.")
		return
	}
	if acc.Amount < cdrCost {
		ctx.Reply(fmt.Sprintf("A cooldown reset costs %s and you have %s.", catsText(cdrCost), catsText(acc.Amount)))
		return
	}

	balance, ok, err := ctx.Cats.ResetCooldown(ctx.Msg.UserID(), cdrCost, catCooldown)
	if err != nil {
		catErr(ctx, err)
		return
	}
	if !ok {
		ctx.Reply("Couldn't reset your cooldown, try again.")
		return
	}
	ctx.Reply(fmt.Sprintf("Cooldown reset for %s. Go catch another! You have %s left.", catsText(cdrCost), catsText(balance)))
}

func catsCmd(ctx *Context) {
	acc, err := ctx.Cats.Get(ctx.Msg.UserID())
	if err != nil {
		catErr(ctx, err)
		return
	}

	reply := fmt.Sprintf("You don't have any cats yet. Try %scat", ctx.Config.Prefix)
	if acc != nil && acc.Amount > 0 {
		reply = fmt.Sprintf("You have %s.", catsText(acc.Amount))
	}
	if left := cooldownLeft(acc); left > 0 {
		reply += fmt.Sprintf(" You can catch another in %s.", utils.FormatLong(left))
	}
	ctx.Reply(reply)
}

func giveCmd(ctx *Context) {
	usage := fmt.Sprintf("Usage: %sgive <user> <amount>", ctx.Config.Prefix)
	if len(ctx.Args) < 2 {
		ctx.Reply(usage)
		return
	}

	amount, err := strconv.ParseInt(ctx.Args[1], 10, 64)
	if err != nil || amount <= 0 {
		ctx.Reply("The amount must be a positive whole number.")
		return
	}

	arg := cleanName(ctx.Args[0])
	if !validLogName.MatchString(arg) {
		ctx.Reply(usage)
		return
	}
	if strings.EqualFold(arg, ctx.Config.Username) {
		ctx.Reply("I don't need cats, keep them.")
		return
	}

	toID, toLogin, err := catTarget(ctx, arg)
	if err != nil {
		if errors.Is(err, utils.ErrTwitchUserNotFound) {
			ctx.Reply(fmt.Sprintf("Couldn't find a Twitch user named %q.", arg))
		} else {
			catErr(ctx, err)
		}
		return
	}
	if toID == ctx.Msg.UserID() {
		ctx.Reply("You can't give cats to yourself.")
		return
	}

	left, err := ctx.Cats.Transfer(ctx.Msg.UserID(), ctx.Msg.User, toID, toLogin, amount)
	if err != nil {
		if errors.Is(err, config.ErrNotEnoughCats) {
			var have int64
			if acc, _ := ctx.Cats.Get(ctx.Msg.UserID()); acc != nil {
				have = acc.Amount
			}
			ctx.Reply(fmt.Sprintf("You only have %s.", catsText(have)))
		} else {
			catErr(ctx, err)
		}
		return
	}
	ctx.Reply(fmt.Sprintf("Gave %s to %s. You have %s left.", catsText(amount), toLogin, catsText(left)))
}

func leaderboardCmd(ctx *Context) {
	top, err := ctx.Cats.Top(leaderboardSize)
	if err != nil {
		catErr(ctx, err)
		return
	}
	if len(top) == 0 {
		ctx.Reply(fmt.Sprintf("Nobody has any cats yet. Be the first with %scat", ctx.Config.Prefix))
		return
	}

	parts := make([]string, 0, len(top))
	for i, a := range top {
		parts = append(parts, fmt.Sprintf("%d. %s (%s)", i+1, a.Login, utils.FormatInt(int(a.Amount))))
	}
	ctx.Reply("Cat leaderboard: " + strings.Join(parts, " • "))
}

func rankCmd(ctx *Context) {
	id, login := ctx.Msg.UserID(), ctx.Msg.User

	if len(ctx.Args) > 0 {
		arg := cleanName(ctx.Args[0])
		if !validLogName.MatchString(arg) {
			ctx.Reply(fmt.Sprintf("Usage: %srank [user]", ctx.Config.Prefix))
			return
		}

		var err error
		id, login, err = catTarget(ctx, arg)
		if err != nil {
			if errors.Is(err, utils.ErrTwitchUserNotFound) {
				ctx.Reply(fmt.Sprintf("Couldn't find a Twitch user named %q.", arg))
			} else {
				catErr(ctx, err)
			}
			return
		}
	}
	self := id == ctx.Msg.UserID()

	acc, err := ctx.Cats.Get(id)
	if err != nil {
		catErr(ctx, err)
		return
	}

	if acc == nil || acc.Amount == 0 {
		if self {
			ctx.Reply(fmt.Sprintf("You have no cats yet. Try %scat", ctx.Config.Prefix))
		} else {
			ctx.Reply(fmt.Sprintf("%s has no cats yet.", login))
		}
		return
	}

	rank, err := ctx.Cats.Rank(acc.Amount)
	if err != nil {
		catErr(ctx, err)
		return
	}
	if self {
		ctx.Reply(fmt.Sprintf("You are rank #%d with %s.", rank, catsText(acc.Amount)))
	} else {
		ctx.Reply(fmt.Sprintf("%s is rank #%d with %s.", login, rank, catsText(acc.Amount)))
	}
}
