package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const apiBase = "https://api.telegram.org/bot"

var httpClient = &http.Client{Timeout: 15 * time.Second}

// maskToken hides the secret part of a bot token for safe logging.
func maskToken(token string) string {
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 || len(parts[1]) < 6 {
		return "***"
	}
	s := parts[1]
	return parts[0] + ":" + s[:3] + "..." + s[len(s)-3:]
}

type botUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type chatInfo struct {
	ID    int64  `json:"id"`
	Type  string `json:"type"`
	Title string `json:"title"`
}

type sendResult struct {
	MessageID int `json:"message_id"`
}

type forumTopicResult struct {
	MessageThreadID int `json:"message_thread_id"`
}

type apiResponse[T any] struct {
	OK          bool   `json:"ok"`
	Result      T      `json:"result"`
	Description string `json:"description"`
}

func doPost[T any](token, method string, body any) (T, error) {
	var zero T
	b, err := json.Marshal(body)
	if err != nil {
		return zero, err
	}
	url := apiBase + token + "/" + method
	resp, err := httpClient.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return zero, fmt.Errorf("telegram %s: %w", method, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var result apiResponse[T]
	if err := json.Unmarshal(raw, &result); err != nil {
		return zero, fmt.Errorf("telegram %s decode: %w", method, err)
	}
	if !result.OK {
		return zero, fmt.Errorf("telegram %s: %s", method, result.Description)
	}
	return result.Result, nil
}

// GetMe validates the token and returns bot info. Safe to log (no token in return).
func GetMe(token string) (*botUser, error) {
	return doPost[*botUser](token, "getMe", struct{}{})
}

// GetChat returns chat info for chatID.
func GetChat(token, chatID string) (*chatInfo, error) {
	return doPost[*chatInfo](token, "getChat", map[string]string{"chat_id": chatID})
}

// SendMessage sends text to chatID, optionally in a forum thread.
func SendMessage(token, chatID, text string, threadID int, parseMode string) (int, error) {
	body := map[string]any{
		"chat_id": chatID,
		"text":    text,
	}
	if parseMode != "" {
		body["parse_mode"] = parseMode
	}
	if threadID > 0 {
		body["message_thread_id"] = threadID
	}
	msg, err := doPost[*sendResult](token, "sendMessage", body)
	if err != nil {
		return 0, err
	}
	return msg.MessageID, nil
}

// CreateForumTopic creates a topic in a supergroup and returns its message_thread_id.
func CreateForumTopic(token, chatID, name, iconEmoji string) (int, error) {
	body := map[string]any{
		"chat_id": chatID,
		"name":    name,
	}
	if iconEmoji != "" {
		body["icon_emoji"] = iconEmoji
	}
	t, err := doPost[*forumTopicResult](token, "createForumTopic", body)
	if err != nil {
		return 0, err
	}
	return t.MessageThreadID, nil
}
