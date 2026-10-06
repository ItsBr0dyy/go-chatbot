package commands

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	utils "github.com/itsbr0dyy/go-chatbot/utils/api"
)

var categoryOrder = []string{"General", "Music", "Twitch info", "Bot management"}

var (
	helpMu   sync.Mutex
	helpLink string
)

func init() {
	Register(&Command{
		Name:     "commands",
		Aliases:  []string{"cmds", "help"},
		Category: "General",
		Usage:    "commands",
		Desc:     "Links to this list.",
		Run:      help,
	})
}

func help(ctx *Context) {
	helpMu.Lock()
	defer helpMu.Unlock()

	if helpLink != "" {
		ctx.Reply("All commands: " + helpLink)
		return
	}

	pasteURL, err := utils.Paste(ctx.Config.PasteAPI, buildHelp(ctx.Config.Prefix))
	if err != nil {
		ctx.Reply(fmt.Sprintf("Couldn't create the command list: %v", err))
		return
	}

	link := pasteURL
	if short, err := utils.Shorten(ctx.Config.ShortAPI, pasteURL); err == nil {
		link = short
	}

	helpLink = link
	ctx.Reply("All commands: " + link)
}

func categoryRank(cat string) int {
	for i, c := range categoryOrder {
		if c == cat {
			return i
		}
	}
	if cat == "Other" {
		return 1000
	}
	return 100
}

func buildHelp(prefix string) string {
	seen := map[*Command]bool{}
	byCat := map[string][]*Command{}
	for _, c := range registry {
		if seen[c] {
			continue
		}
		seen[c] = true

		cat := c.Category
		if cat == "" {
			cat = "Other"
		}
		byCat[cat] = append(byCat[cat], c)
	}

	cats := make([]string, 0, len(byCat))
	for c := range byCat {
		cats = append(cats, c)
	}
	sort.Slice(cats, func(i, j int) bool {
		ri, rj := categoryRank(cats[i]), categoryRank(cats[j])
		if ri != rj {
			return ri < rj
		}
		return cats[i] < cats[j]
	})

	var b strings.Builder
	fmt.Fprintf(&b, "Commands (prefix: %s)\n", prefix)
	b.WriteString("Access: Everyone = anyone in chat, Helper = helpers and owners, Owner = owners only\n")

	for _, cat := range cats {
		cmds := byCat[cat]
		sort.Slice(cmds, func(i, j int) bool { return cmds[i].Name < cmds[j].Name })

		fmt.Fprintf(&b, "\n== %s ==\n", cat)
		for _, c := range cmds {
			fmt.Fprintf(&b, "\n%s%s\n", prefix, c.Name)

			if c.Desc != "" {
				fmt.Fprintf(&b, "  %s\n", c.Desc)
			}
			usage := c.Usage
			if usage == "" {
				usage = c.Name
			}
			fmt.Fprintf(&b, "  Usage: %s%s\n", prefix, usage)

			if len(c.Aliases) > 0 {
				aliases := make([]string, len(c.Aliases))
				for n, a := range c.Aliases {
					aliases[n] = prefix + a
				}
				fmt.Fprintf(&b, "  Aliases: %s\n", strings.Join(aliases, ", "))
			}
			fmt.Fprintf(&b, "  Access: %s\n", c.Access)
		}
	}
	return b.String()
}
