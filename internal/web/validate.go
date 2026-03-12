package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

// ValidateModelKey validates an API key by calling the provider's models endpoint.
func ValidateModelKey(provider, apiKey, model string) error {
	switch provider {
	case "openai":
		return validateOpenAI(apiKey)
	case "anthropic":
		return validateAnthropic(apiKey)
	case "deepseek":
		return validateDeepSeek(apiKey)
	case "google":
		return validateGoogle(apiKey)
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}
}

func validateOpenAI(apiKey string) error {
	req, err := http.NewRequest("GET", "https://api.openai.com/v1/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return doValidationRequest(req, "OpenAI")
}

func validateAnthropic(apiKey string) error {
	req, err := http.NewRequest("GET", "https://api.anthropic.com/v1/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	return doValidationRequest(req, "Anthropic")
}

func validateDeepSeek(apiKey string) error {
	req, err := http.NewRequest("GET", "https://api.deepseek.com/v1/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	return doValidationRequest(req, "DeepSeek")
}

func validateGoogle(apiKey string) error {
	req, err := http.NewRequest("GET", "https://generativelanguage.googleapis.com/v1/models?key="+apiKey, nil)
	if err != nil {
		return err
	}
	return doValidationRequest(req, "Google")
}

func doValidationRequest(req *http.Request, provider string) error {
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%s API request failed: %w", provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}
	return fmt.Errorf("%s returned HTTP %d: %s", provider, resp.StatusCode, string(body))
}

// ValidateChannelToken validates a channel bot token.
func ValidateChannelToken(channel, token, appID, appSecret string) error {
	switch channel {
	case "telegram":
		return validateTelegram(token)
	case "discord":
		return validateDiscord(token)
	case "slack":
		return validateSlack(token)
	case "lark":
		return validateLark(appID, appSecret)
	default:
		return fmt.Errorf("unsupported channel: %s", channel)
	}
}

func validateTelegram(token string) error {
	req, err := http.NewRequest("GET", "https://api.telegram.org/bot"+token+"/getMe", nil)
	if err != nil {
		return err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Telegram API request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse Telegram response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("invalid Telegram bot token")
	}
	return nil
}

func validateDiscord(token string) error {
	req, err := http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+token)
	return doValidationRequest(req, "Discord")
}

// resolveBotName calls the channel platform API to get the bot's display name.
// Returns empty string on failure (non-critical — text @mention detection
// simply won't work, but the bot still functions normally).
func resolveBotName(channel, token string) string {
	switch channel {
	case "discord":
		return resolveDiscordBotName(token)
	case "slack":
		return resolveSlackBotName(token)
	default:
		return ""
	}
}

func resolveDiscordBotName(token string) string {
	req, err := http.NewRequest("GET", "https://discord.com/api/v10/users/@me", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bot "+token)
	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()
	var result struct {
		GlobalName string `json:"global_name"`
		Username   string `json:"username"`
	}
	if json.NewDecoder(resp.Body).Decode(&result) != nil {
		return ""
	}
	if result.GlobalName != "" {
		return result.GlobalName
	}
	return result.Username
}

func resolveSlackBotName(token string) string {
	req, err := http.NewRequest("POST", "https://slack.com/api/auth.test", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()
	var result struct {
		OK   bool   `json:"ok"`
		User string `json:"user"`
	}
	if json.NewDecoder(resp.Body).Decode(&result) != nil || !result.OK {
		return ""
	}
	return result.User
}

func validateSlack(token string) error {
	req, err := http.NewRequest("POST", "https://slack.com/api/auth.test", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Slack API request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse Slack response: %w", err)
	}
	if !result.OK {
		return fmt.Errorf("invalid Slack token: %s", result.Error)
	}
	return nil
}

func validateLark(appID, appSecret string) error {
	if appID == "" || appSecret == "" {
		return fmt.Errorf("App ID and App Secret are required for Lark")
	}
	body := fmt.Sprintf(`{"app_id":"%s","app_secret":"%s"}`, appID, appSecret)
	req, err := http.NewRequest("POST", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		strings.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Lark API request failed: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to parse Lark response: %w", err)
	}
	if result.Code != 0 {
		return fmt.Errorf("Lark validation failed: %s", result.Msg)
	}
	return nil
}
