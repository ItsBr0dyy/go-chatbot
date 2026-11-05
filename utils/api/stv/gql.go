package stv

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
)

const gqlURL = "https://7tv.io/v4/gql"

type GQLError struct {
	Message string
	Code    string
}

func (e *GQLError) Error() string {
	if e.Code != "" {
		return e.Code + ": " + e.Message
	}
	return e.Message
}

func isAuthError(msg, code string) bool {
	s := strings.ToLower(msg + " " + code)
	for _, k := range []string{"unauthenticated", "not authenticated", "token", "expired", "not logged in"} {
		if strings.Contains(s, k) {
			return true
		}
	}
	return false
}

func (c *Client) do(query string, vars map[string]any, out any) error {
	payload, err := json.Marshal(map[string]any{"query": query, "variables": vars})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, gqlURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return ErrTokenInvalid
	}

	var env struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message    string         `json:"message"`
			Extensions map[string]any `json:"extensions"`
		} `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("7tv returned %d", resp.StatusCode)
		}
		return err
	}

	if len(env.Errors) > 0 {
		e := env.Errors[0]
		code, _ := e.Extensions["code"].(string)
		if isAuthError(e.Message, code) {
			return ErrTokenInvalid
		}
		return &GQLError{Message: e.Message, Code: code}
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("7tv returned %d", resp.StatusCode)
	}

	if out != nil && len(env.Data) > 0 {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

func (c *Client) UserIDFromTwitch(twitchID string) (string, error) {
	const query = `
query GetUserByTwitch($platformId: String!) {
  users {
    userByConnection(platform: TWITCH, platformId: $platformId) {
      id
    }
  }
}`

	var out struct {
		Users struct {
			UserByConnection *struct {
				ID string `json:"id"`
			} `json:"userByConnection"`
		} `json:"users"`
	}
	if err := c.do(query, map[string]any{"platformId": twitchID}, &out); err != nil {
		var ge *GQLError
		if errors.As(err, &ge) && strings.Contains(strings.ToLower(ge.Message), "not found") {
			return "", ErrUserNotFound
		}
		return "", err
	}
	if out.Users.UserByConnection == nil || out.Users.UserByConnection.ID == "" {
		return "", ErrUserNotFound
	}
	return out.Users.UserByConnection.ID, nil
}

type SetEmote struct {
	ID    string  `json:"id"`
	Alias *string `json:"alias"`
	Emote struct {
		ID          string `json:"id"`
		DefaultName string `json:"defaultName"`
	} `json:"emote"`
}

func (e SetEmote) Name() string {
	if e.Alias != nil && *e.Alias != "" {
		return *e.Alias
	}
	return e.Emote.DefaultName
}

type EmoteSet struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Capacity int    `json:"capacity"`
	Emotes   struct {
		TotalCount int        `json:"totalCount"`
		Items      []SetEmote `json:"items"`
	} `json:"emotes"`
}

type UserSet struct {
	Set     *EmoteSet
	Editors []string
}

func (c *Client) ActiveSet(userID string) (*UserSet, error) {
	const query = `
query GetActiveSet($id: Id!) {
  users {
    user(id: $id) {
      style {
        activeEmoteSet {
          id
          name
          capacity
          emotes {
            totalCount
            items {
              id
              alias
              emote {
                defaultName
                id
              }
            }
          }
        }
      }
      editors {
        editorId
        addedAt
      }
    }
  }
}`

	var out struct {
		Users struct {
			User *struct {
				Style struct {
					ActiveEmoteSet *EmoteSet `json:"activeEmoteSet"`
				} `json:"style"`
				Editors []struct {
					EditorID string `json:"editorId"`
				} `json:"editors"`
			} `json:"user"`
		} `json:"users"`
	}
	if err := c.do(query, map[string]any{"id": userID}, &out); err != nil {
		return nil, err
	}

	res := &UserSet{}
	if u := out.Users.User; u != nil {
		res.Set = u.Style.ActiveEmoteSet
		for _, e := range u.Editors {
			res.Editors = append(res.Editors, e.EditorID)
		}
	}
	return res, nil
}

type EmoteResult struct {
	ID          string `json:"id"`
	DefaultName string `json:"defaultName"`
}

func (c *Client) SearchEmotes(q string) ([]EmoteResult, error) {
	const query = `
query SearchEmotes($query: String!) {
  emotes {
    search(sort: {sortBy: UPLOAD_DATE, order: ASCENDING}, query: $query) {
      items {
        id
        defaultName
      }
    }
  }
}`

	var out struct {
		Emotes struct {
			Search struct {
				Items []EmoteResult `json:"items"`
			} `json:"search"`
		} `json:"emotes"`
	}
	if err := c.do(query, map[string]any{"query": q}, &out); err != nil {
		return nil, err
	}
	return out.Emotes.Search.Items, nil
}

func (c *Client) AddEmote(setID, emoteID string, alias *string) error {
	const query = `
mutation AddEmote($setID: Id!, $emote: EmoteSetEmoteId!) {
  emoteSets {
    emoteSet(id: $setID) {
      addEmote(id: $emote) { id }
    }
  }
}`
	vars := map[string]any{
		"setID": setID,
		"emote": map[string]any{"emoteId": emoteID, "alias": alias},
	}
	return c.do(query, vars, nil)
}

func (c *Client) RemoveEmote(setID, emoteID string, alias *string) error {
	const query = `
mutation RemoveEmote($setID: Id!, $emote: EmoteSetEmoteId!) {
  emoteSets {
    emoteSet(id: $setID) {
      removeEmote(id: $emote) { id }
    }
  }
}`
	vars := map[string]any{
		"setID": setID,
		"emote": map[string]any{"emoteId": emoteID, "alias": alias},
	}
	return c.do(query, vars, nil)
}

func (c *Client) UpdateEmoteAlias(setID, emoteID, alias string) error {
	const query = `
mutation UpdateEmoteAlias($setID: Id!, $emoteID: EmoteSetEmoteId!, $alias: String!) {
  emoteSets {
    emoteSet(id: $setID) {
      updateEmoteAlias(id: $emoteID, alias: $alias) {
        id
        alias
      }
    }
  }
}`
	vars := map[string]any{
		"setID":   setID,
		"emoteID": map[string]any{"emoteId": emoteID},
		"alias":   alias,
	}
	return c.do(query, vars, nil)
}

var ulidPattern = regexp.MustCompile(`^[0-9A-Za-z]{26}$`)

func (c *Client) PendingEditorRequests(userID string) ([]string, error) {
	if !ulidPattern.MatchString(userID) {
		return nil, errors.New("invalid 7tv user id")
	}

	query := fmt.Sprintf(`
query EditorCheck {
  users {
    user(id: %q) {
      editorFor {
        state
        userId
      }
    }
  }
}`, userID)

	var out struct {
		Users struct {
			User *struct {
				EditorFor []struct {
					State  string `json:"state"`
					UserID string `json:"userId"`
				} `json:"editorFor"`
			} `json:"user"`
		} `json:"users"`
	}
	if err := c.do(query, map[string]any{}, &out); err != nil {
		return nil, err
	}

	var pending []string
	if out.Users.User != nil {
		for _, e := range out.Users.User.EditorFor {
			if e.State == "PENDING" {
				pending = append(pending, e.UserID)
			}
		}
	}
	return pending, nil
}

func (c *Client) AcceptEditorRequest(userID, requesterID string) error {
	if !ulidPattern.MatchString(userID) || !ulidPattern.MatchString(requesterID) {
		return errors.New("invalid 7tv user id")
	}

	query := fmt.Sprintf(`
mutation AcceptEditor {
  userEditors {
    editor(editorId: %q, userId: %q) {
      updateState(state: ACCEPT) {
        state
      }
    }
  }
}`, userID, requesterID)

	return c.do(query, map[string]any{}, nil)
}
