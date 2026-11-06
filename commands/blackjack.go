package commands

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"time"
)

const bjTimeout = 2 * time.Minute

// bjSuits given by claude of course im too lazy to search
var (
	bjRanks = []string{"A", "2", "3", "4", "5", "6", "7", "8", "9", "10", "J", "Q", "K"}
	bjSuits = []string{"♠", "♥", "♦", "♣"}
)

type bjGame struct {
	bet    int64
	deck   []int
	player []int
	dealer []int
	last   time.Time
	ready  bool
}

var (
	bjMu    sync.Mutex
	bjGames = map[string]*bjGame{}
)

func init() {
	Register(&Command{
		Name:     "bj",
		Aliases:  []string{"blackjack"},
		Category: "Cats",
		Usage:    "bj <amount|all|half>",
		Desc:     "Play blackjack against the dealer. Then use hit or stand. Blackjack pays 3:2.",
		Run:      bjCmd,
	})
	Register(&Command{
		Name:     "hit",
		Category: "Cats",
		Usage:    "hit",
		Desc:     "Take another card in your blackjack game.",
		Run:      hitCmd,
	})
	Register(&Command{
		Name:     "stand",
		Category: "Cats",
		Usage:    "stand",
		Desc:     "Stop drawing and let the dealer play.",
		Run:      standCmd,
	})
}

func cardName(c int) string { return bjRanks[c%13] + bjSuits[c/13] }

func handText(cards []int) string {
	names := make([]string, len(cards))
	for i, c := range cards {
		names[i] = cardName(c)
	}
	return "[" + strings.Join(names, " ") + "]"
}

func handValue(cards []int) int {
	total, aces := 0, 0
	for _, c := range cards {
		switch r := c % 13; {
		case r == 0:
			total += 11
			aces++
		case r >= 9: // 10, J, Q, K
			total += 10
		default:
			total += r + 1
		}
	}
	for total > 21 && aces > 0 {
		total -= 10
		aces--
	}
	return total
}

func isNatural(cards []int) bool { return len(cards) == 2 && handValue(cards) == 21 }

func (g *bjGame) draw() int {
	c := g.deck[0]
	g.deck = g.deck[1:]
	return c
}

func newBJGame(bet int64) *bjGame {
	g := &bjGame{bet: bet, deck: rand.Perm(52), last: time.Now()}
	g.player = []int{g.draw(), g.draw()}
	g.dealer = []int{g.draw(), g.draw()}
	return g
}

func bjLookup(uid string) (g *bjGame, timedOut bool) {
	g, ok := bjGames[uid]
	if !ok {
		return nil, false
	}
	if time.Since(g.last) > bjTimeout {
		delete(bjGames, uid)
		return nil, true
	}
	return g, false
}

func bjNoGame(ctx *Context, timedOut bool) {
	if timedOut {
		ctx.Reply(fmt.Sprintf("Your blackjack game timed out and the bet was lost. Start a new one with %sbj <amount>", ctx.Config.Prefix))
		return
	}
	ctx.Reply(fmt.Sprintf("You don't have a game running. Start one with %sbj <amount>", ctx.Config.Prefix))
}

func bjFinish(ctx *Context, g *bjGame, payout int64, text string) {
	balance, ok := settle(ctx, 0, payout)
	if !ok {
		return
	}

	switch {
	case payout > g.bet:
		ctx.Reply(fmt.Sprintf("%s You won %s! You have %s now.", text, catsText(payout-g.bet), catsText(balance)))
	case payout == g.bet:
		ctx.Reply(fmt.Sprintf("%s Push, you get your %s back. You have %s.", text, catsText(g.bet), catsText(balance)))
	default:
		ctx.Reply(fmt.Sprintf("%s You lost %s. You have %s left.", text, catsText(g.bet), catsText(balance)))
	}
}

func bjResolve(ctx *Context, g *bjGame, intro string) {
	for handValue(g.dealer) < 17 {
		g.dealer = append(g.dealer, g.draw())
	}
	p, d := handValue(g.player), handValue(g.dealer)
	text := fmt.Sprintf("%s Dealer has %s (%d).", intro, handText(g.dealer), d)

	switch {
	case d > 21:
		bjFinish(ctx, g, g.bet*2, text+" Dealer busts!")
	case p > d:
		bjFinish(ctx, g, g.bet*2, text)
	case p == d:
		bjFinish(ctx, g, g.bet, text)
	default:
		bjFinish(ctx, g, 0, text)
	}
}

func bjCmd(ctx *Context) {
	usage := fmt.Sprintf("Usage: %sbj <amount|all|half>", ctx.Config.Prefix)
	if len(ctx.Args) < 1 {
		ctx.Reply(usage)
		return
	}
	bet, ok := readBet(ctx, ctx.Args[0], usage)
	if !ok {
		return
	}

	uid := ctx.Msg.UserID()
	prefix := ctx.Config.Prefix

	bjMu.Lock()
	if cur, _ := bjLookup(uid); cur != nil {
		text := fmt.Sprintf("You already have a game running: %s (%d), dealer shows %s. %shit or %sstand",
			handText(cur.player), handValue(cur.player), cardName(cur.dealer[0]), prefix, prefix)
		bjMu.Unlock()
		ctx.Reply(text)
		return
	}
	g := newBJGame(bet)
	bjGames[uid] = g
	bjMu.Unlock()

	if _, ok := settle(ctx, bet, 0); !ok {
		bjMu.Lock()
		if bjGames[uid] == g {
			delete(bjGames, uid)
		}
		bjMu.Unlock()
		return
	}

	playerBJ, dealerBJ := isNatural(g.player), isNatural(g.dealer)

	bjMu.Lock()
	if playerBJ || dealerBJ {
		delete(bjGames, uid)
	} else {
		g.ready = true
		g.last = time.Now()
	}
	bjMu.Unlock()

	switch {
	case playerBJ && dealerBJ:
		bjFinish(ctx, g, g.bet, fmt.Sprintf("You and the dealer both have blackjack: %s and %s.", handText(g.player), handText(g.dealer)))
	case playerBJ:
		bjFinish(ctx, g, g.bet*5/2, fmt.Sprintf("Blackjack! %s", handText(g.player)))
	case dealerBJ:
		bjFinish(ctx, g, 0, fmt.Sprintf("The dealer has blackjack: %s.", handText(g.dealer)))
	default:
		ctx.Reply(fmt.Sprintf("Blackjack for %s: you have %s (%d), dealer shows %s. %shit or %sstand",
			catsText(bet), handText(g.player), handValue(g.player), cardName(g.dealer[0]), prefix, prefix))
	}
}

func hitCmd(ctx *Context) {
	uid := ctx.Msg.UserID()
	prefix := ctx.Config.Prefix

	bjMu.Lock()
	g, timedOut := bjLookup(uid)
	if g == nil || !g.ready {
		bjMu.Unlock()
		if g != nil {
			ctx.Reply("Hold on, your game is still being dealt.")
		} else {
			bjNoGame(ctx, timedOut)
		}
		return
	}

	card := g.draw()
	g.player = append(g.player, card)
	g.last = time.Now()
	total := handValue(g.player)
	if total >= 21 {
		delete(bjGames, uid)
	}
	drew := fmt.Sprintf("You drew %s: %s (%d).", cardName(card), handText(g.player), total)
	bjMu.Unlock()

	switch {
	case total > 21:
		bjFinish(ctx, g, 0, drew+" Bust!")
	case total == 21:
		bjResolve(ctx, g, drew)
	default:
		ctx.Reply(fmt.Sprintf("%s %shit or %sstand", drew, prefix, prefix))
	}
}

func standCmd(ctx *Context) {
	uid := ctx.Msg.UserID()

	bjMu.Lock()
	g, timedOut := bjLookup(uid)
	if g == nil || !g.ready {
		bjMu.Unlock()
		if g != nil {
			ctx.Reply("Hold on, your game is still being dealt.")
		} else {
			bjNoGame(ctx, timedOut)
		}
		return
	}
	delete(bjGames, uid)
	intro := fmt.Sprintf("You stand on %d.", handValue(g.player))
	bjMu.Unlock()

	bjResolve(ctx, g, intro)
}
