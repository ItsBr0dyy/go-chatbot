package loop

import (
	"log"
	"time"

	"github.com/itsbr0dyy/go-chatbot/config"
	"github.com/itsbr0dyy/go-chatbot/utils"
	"github.com/itsbr0dyy/go-chatbot/utils/api/stv"
)

const editorCheckInterval = 10 * time.Second

func StartEditorAccepter(cfg *config.Config) {
	if cfg.STVToken == "" {
		return
	}
	c := stv.New(cfg.STVToken)

	var (
		myID    string
		lastErr string
	)

	logOnce := func(err error) {
		if msg := err.Error(); msg != lastErr {
			log.Printf("7tv editor accepter: %v", err)
			lastErr = msg
		}
	}

	Every(editorCheckInterval, func() {
		if myID == "" {
			id, err := ownSTVID(cfg, c)
			if err != nil {
				logOnce(err)
				return
			}
			myID = id
		}

		pending, err := c.PendingEditorRequests(myID)
		if err != nil {
			logOnce(err)
			return
		}
		lastErr = ""

		for _, requester := range pending {
			if err := c.AcceptEditorRequest(myID, requester); err != nil {
				logOnce(err)
				continue
			}
			log.Printf("7tv: accepted editor request from %s", requester)
		}
	})
}

func ownSTVID(cfg *config.Config, c *stv.Client) (string, error) {
	if cfg.STVUserID != "" {
		return cfg.STVUserID, nil
	}
	u, err := utils.IVRUser(cfg.Username)
	if err != nil {
		return "", err
	}
	return c.UserIDFromTwitch(u.ID)
}
