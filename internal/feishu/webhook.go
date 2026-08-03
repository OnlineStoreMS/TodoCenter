package feishu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{httpClient: &http.Client{Timeout: 15 * time.Second}}
}

func Sign(secret string, timestamp int64) (string, error) {
	if secret == "" {
		return "", nil
	}
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, secret)
	mac := hmac.New(sha256.New, []byte(stringToSign))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (c *Client) SendText(ctx context.Context, webhookURL, secret, text string) error {
	return c.postSignedJSON(ctx, webhookURL, secret, map[string]any{
		"msg_type": "text",
		"content":  map[string]any{"text": text},
	})
}

type InteractiveCard struct {
	Title    string
	Template string // blue | orange | red | green
	Markdown string
}

func (c *Client) SendInteractiveCard(ctx context.Context, webhookURL, secret string, card InteractiveCard) error {
	if card.Title == "" {
		card.Title = "待办通知"
	}
	if card.Template == "" {
		card.Template = "blue"
	}
	return c.postSignedJSON(ctx, webhookURL, secret, map[string]any{
		"msg_type": "interactive",
		"card": map[string]any{
			"schema": "2.0",
			"config": map[string]any{"update_multi": true},
			"header": map[string]any{
				"title":    map[string]any{"tag": "plain_text", "content": card.Title},
				"template": card.Template,
			},
			"body": map[string]any{
				"direction": "vertical",
				"elements": []any{
					map[string]any{
						"tag":        "markdown",
						"content":    card.Markdown,
						"text_align": "left",
					},
				},
			},
		},
	})
}

func (c *Client) postSignedJSON(ctx context.Context, webhookURL, secret string, payload map[string]any) error {
	webhookURL = strings.TrimSpace(webhookURL)
	if webhookURL == "" {
		return fmt.Errorf("webhook url is required")
	}
	ts := time.Now().Unix()
	payload["timestamp"] = strconv.FormatInt(ts, 10)
	if secret = strings.TrimSpace(secret); secret != "" {
		sign, err := Sign(secret, ts)
		if err != nil {
			return err
		}
		payload["sign"] = sign
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := string(raw)
		if len(msg) > 200 {
			msg = msg[:200] + "..."
		}
		return fmt.Errorf("feishu webhook http %d: %s", resp.StatusCode, msg)
	}
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &result)
	}
	if result.Code != 0 && result.Msg != "" {
		return fmt.Errorf("feishu webhook: %s", result.Msg)
	}
	return nil
}
