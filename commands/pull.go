package commands

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	pullMu   sync.Mutex
	urlCreds = regexp.MustCompile(`://[^@/\s]+@`)
)

func init() {
	Register(&Command{
		Name:     "pull",
		Category: "Bot management",
		Usage:    "pull",
		Desc:     "Pulls the latest code from origin main, checks it builds, and restarts the bot.",
		Access:   AccessOwner,
		Run:      pullCmd,
	})
}

func execOutput(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func logStep(step, out string, err error) {
	log.Printf("pull: %s failed: %v\n%s", step, err, urlCreds.ReplaceAllString(out, "://***@"))
}

func pullCmd(ctx *Context) {
	if !pullMu.TryLock() {
		ctx.Reply("A pull is already running.")
		return
	}
	defer pullMu.Unlock()

	old, err := execOutput(10*time.Second, "git", "rev-parse", "--short", "HEAD")
	if err != nil {
		logStep("rev-parse", old, err)
		ctx.Reply("Couldn't read the git state. Is this folder a git repository?")
		return
	}

	if out, err := execOutput(2*time.Minute, "git", "pull", "--ff-only", "origin", "main"); err != nil {
		logStep("git pull", out, err)
		ctx.Reply("git pull failed, check the logs.")
		return
	}

	cur, err := execOutput(10*time.Second, "git", "rev-parse", "--short", "HEAD")
	if err != nil {
		logStep("rev-parse", cur, err)
		ctx.Reply("Pulled, but couldn't read the new version. Check the logs.")
		return
	}
	if cur == old {
		ctx.Reply("Already up to date.")
		return
	}

	count, _ := execOutput(10*time.Second, "git", "rev-list", "--count", old+".."+cur)
	ctx.Reply(fmt.Sprintf("Pulled %s new commits (%s → %s), checking that it builds...", count, old, cur))

	if out, err := execOutput(10*time.Minute, "go", "build", "-o", os.DevNull, "."); err != nil {
		logStep("build", out, err)

		if rb, rerr := execOutput(30*time.Second, "git", "reset", "--keep", old); rerr != nil {
			logStep("rollback", rb, rerr)
			ctx.Reply("The new code doesn't build and I couldn't roll back. Check the logs.")
		} else {
			ctx.Reply(fmt.Sprintf("The new code doesn't build, rolled back to %s. Check the logs.", old))
		}
		return
	}

	ctx.Reply("Build OK, restarting.")

	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()
}
