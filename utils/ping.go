package utils

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

var (
	pingMu  sync.Mutex
	pending = map[string]chan struct{}{}
	pingSeq atomic.Int64
)

func (c *Client) Ping() (time.Duration, error) {
	token := fmt.Sprintf("latency-%d", pingSeq.Add(1))
	ch := make(chan struct{}, 1)

	pingMu.Lock()
	pending[token] = ch
	pingMu.Unlock()
	defer func() {
		pingMu.Lock()
		delete(pending, token)
		pingMu.Unlock()
	}()

	start := time.Now()
	if err := c.Send("PING :" + token); err != nil {
		return 0, err
	}

	select {
	case <-ch:
		return time.Since(start), nil
	case <-time.After(5 * time.Second):
		return 0, errors.New("ping timed out")
	}
}

func HandlePong(token string) {
	pingMu.Lock()
	ch, ok := pending[token]
	pingMu.Unlock()
	if ok {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
