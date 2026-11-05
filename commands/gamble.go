package commands

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"

	"github.com/itsbr0dyy/go-chatbot/config"
)

const (
	maxBet        = 500
	flipWinChance = 49
	slotsJackpot  = 20
)

// asked claude for these chillll
var slotSymbols = []string{"🐱", "🐟", "🧶", "🥛", "🐭", "🐾"}

func init() {
	Register(&Command{
		Name:     "gamble",
		Category: "Cats",
		Usage:    "gamble <amount|all|half>",
		Desc:     "Roll the dice with your cats. Double up on a good roll, lose the bet on a bad one.",
		Run:      gambleCmd,
	})
	Register(&Command{
		Name:     "slots",
		Category: "Cats",
		Usage:    "slots <amount|all|half>",
		Desc:     "Spin the slot machine. Two of a kind gives your cats back, three pays big.",
		Run:      slotsCmd,
	})
	Register(&Command{
		Name:     "flip",
		Category: "Cats",
		Usage:    "flip <heads|tails> <amount|all|half>",
		Desc:     "Call a coin flip. Win and you double your bet.",
		Run:      flipCmd,
	})
}

func readBet(ctx *Context, arg, usage string) (int64, bool) {
	acc, err := ctx.Cats.Get(ctx.Msg.UserID())
	if err != nil {
		catErr(ctx, err)
		return 0, false
	}

	var have int64
	if acc != nil {
		have = acc.Amount
	}
	if have == 0 {
		ctx.Reply(fmt.Sprintf("You don't have any cats to gamble. Try %scat", ctx.Config.Prefix))
		return 0, false
	}

	var bet int64
	switch strings.ToLower(arg) {
	case "all":
		bet = min(have, maxBet)
	case "half":
		bet = min(max(have/2, 1), maxBet)
	default:
		n, err := strconv.ParseInt(arg, 10, 64)
		if err != nil || n <= 0 {
			ctx.Reply(usage)
			return 0, false
		}
		if n > maxBet {
			ctx.Reply(fmt.Sprintf("The max bet is %s.", catsText(maxBet)))
			return 0, false
		}
		bet = n
	}

	if bet > have {
		ctx.Reply(fmt.Sprintf("You only have %s.", catsText(have)))
		return 0, false
	}
	return bet, true
}

func settle(ctx *Context, bet, payout int64) (int64, bool) {
	balance, err := ctx.Cats.Settle(ctx.Msg.UserID(), ctx.Msg.User, bet, payout)
	if err != nil {
		if errors.Is(err, config.ErrNotEnoughCats) {
			ctx.Reply("You don't have enough cats for that bet.")
		} else {
			catErr(ctx, err)
		}
		return 0, false
	}
	return balance, true
}

func gambleCmd(ctx *Context) {
	usage := fmt.Sprintf("Usage: %sgamble <amount|all|half>", ctx.Config.Prefix)
	if len(ctx.Args) < 1 {
		ctx.Reply(usage)
		return
	}
	bet, ok := readBet(ctx, ctx.Args[0], usage)
	if !ok {
		return
	}

	roll := rand.IntN(100) + 1

	var payout int64
	verdict := ""
	switch {
	case roll <= 55:
		payout = 0
	case roll <= 94:
		payout = bet * 2
	case roll <= 99:
		payout = bet * 3
		verdict = "Nice roll! "
	default:
		payout = bet * 5
		verdict = "JACKPOT! "
	}

	balance, ok := settle(ctx, bet, payout)
	if !ok {
		return
	}

	if payout == 0 {
		ctx.Reply(fmt.Sprintf("You rolled %d and lost %s. You have %s left.",
			roll, catsText(bet), catsText(balance)))
		return
	}
	ctx.Reply(fmt.Sprintf("%sYou rolled %d and won %s! You have %s now.",
		verdict, roll, catsText(payout-bet), catsText(balance)))
}

func slotsCmd(ctx *Context) {
	usage := fmt.Sprintf("Usage: %sslots <amount|all|half>", ctx.Config.Prefix)
	if len(ctx.Args) < 1 {
		ctx.Reply(usage)
		return
	}
	bet, ok := readBet(ctx, ctx.Args[0], usage)
	if !ok {
		return
	}

	reels := [3]string{pick(slotSymbols), pick(slotSymbols), pick(slotSymbols)}
	display := "[ " + strings.Join(reels[:], " ") + " ]"

	var payout int64
	switch {
	case reels[0] == reels[1] && reels[1] == reels[2]:
		payout = bet * slotsJackpot
	case reels[0] == reels[1] || reels[1] == reels[2] || reels[0] == reels[2]:
		payout = bet
	}

	balance, ok := settle(ctx, bet, payout)
	if !ok {
		return
	}

	switch {
	case payout > bet:
		ctx.Reply(fmt.Sprintf("%s JACKPOT! You won %s! You have %s now.",
			display, catsText(payout-bet), catsText(balance)))
	case payout == bet:
		ctx.Reply(fmt.Sprintf("%s Two of a kind, you get your cats back. You have %s.",
			display, catsText(balance)))
	default:
		ctx.Reply(fmt.Sprintf("%s No match, you lost %s. You have %s left.",
			display, catsText(bet), catsText(balance)))
	}
}

func flipCmd(ctx *Context) {
	usage := fmt.Sprintf("Usage: %sflip <heads|tails> <amount|all|half>", ctx.Config.Prefix)
	if len(ctx.Args) < 2 {
		ctx.Reply(usage)
		return
	}

	var calledHeads bool
	switch strings.ToLower(ctx.Args[0]) {
	case "heads", "head", "h":
		calledHeads = true
	case "tails", "tail", "t":
		calledHeads = false
	default:
		ctx.Reply(usage)
		return
	}

	bet, ok := readBet(ctx, ctx.Args[1], usage)
	if !ok {
		return
	}

	won := rand.IntN(100) < flipWinChance
	landedHeads := calledHeads == won

	side := "tails"
	if landedHeads {
		side = "heads"
	}

	var payout int64
	if won {
		payout = bet * 2
	}

	balance, ok := settle(ctx, bet, payout)
	if !ok {
		return
	}

	if won {
		ctx.Reply(fmt.Sprintf("The coin landed on %s. You won %s! You have %s now.",
			side, catsText(bet), catsText(balance)))
	} else {
		ctx.Reply(fmt.Sprintf("The coin landed on %s. You lost %s. You have %s left.",
			side, catsText(bet), catsText(balance)))
	}
}
