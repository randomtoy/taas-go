package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/randomtoy/taas-go/internal/domain"
	"github.com/randomtoy/taas-go/internal/ports"
)

// Client implements ports.Interpreter via the OpenRouter API.
type Client struct {
	httpClient     *http.Client
	apiKey         string
	baseURL        string
	model          string
	fallbackModels []string
	logger         *slog.Logger
}

func NewClient(httpClient *http.Client, apiKey, baseURL, model string, fallbackModels []string, logger *slog.Logger) *Client {
	return &Client{
		httpClient:     httpClient,
		apiKey:         apiKey,
		baseURL:        strings.TrimRight(baseURL, "/"),
		model:          model,
		fallbackModels: fallbackModels,
		logger:         logger,
	}
}

// chatRequest / chatResponse mirror the OpenAI-compatible API shapes.
type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (c *Client) Interpret(ctx context.Context, in ports.InterpretInput) (ports.InterpretOutput, error) {
	models := make([]string, 0, 1+len(c.fallbackModels))
	models = append(models, c.model)
	models = append(models, c.fallbackModels...)

	var lastErr error
	for _, model := range models {
		out, err := c.interpretWithModel(ctx, in, model)
		if err == nil {
			return out, nil
		}
		lastErr = err
		if len(models) > 1 {
			c.logger.WarnContext(ctx, "model failed, trying next", "model", model, "error", err)
		}
	}

	return ports.InterpretOutput{}, lastErr
}

func (c *Client) interpretWithModel(ctx context.Context, in ports.InterpretInput, model string) (ports.InterpretOutput, error) {
	systemPrompt := buildSystemPrompt(in.Lang)
	userPrompt := buildUserPrompt(in)

	content, err := c.callLLM(ctx, model, systemPrompt, userPrompt)
	if err != nil {
		return ports.InterpretOutput{}, fmt.Errorf("%w: %w", domain.ErrUpstreamLLM, err)
	}

	var out ports.InterpretOutput
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		c.logger.WarnContext(ctx, "LLM returned invalid JSON, retrying", "model", model, "error", err)
		content, err = c.callLLM(ctx, model, systemPrompt, retryPrompt(content))
		if err != nil {
			return ports.InterpretOutput{}, fmt.Errorf("%w: %w", domain.ErrUpstreamLLM, err)
		}
		if err := json.Unmarshal([]byte(content), &out); err != nil {
			return ports.InterpretOutput{}, fmt.Errorf("%w: %w", domain.ErrInvalidLLMJSON, err)
		}
	}

	if out.Style == "" {
		out.Style = "neutral"
	}
	if out.Disclaimer == "" {
		out.Disclaimer = "For reflection/entertainment; not medical/legal/financial advice."
	}
	out.Model = model

	return out, nil
}

func (c *Client) callLLM(ctx context.Context, model, system, user string) (string, error) {
	reqBody := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := c.baseURL + "/chat/completions"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("upstream status %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return strings.TrimSpace(chatResp.Choices[0].Message.Content), nil
}

// langNames maps common BCP 47 codes to human-readable language names.
var langNames = map[string]string{
	"en": "English",
	"ru": "Russian",
	"es": "Spanish",
	"fr": "French",
	"de": "German",
	"it": "Italian",
	"pt": "Portuguese",
	"ja": "Japanese",
	"ko": "Korean",
	"zh": "Chinese",
	"ar": "Arabic",
	"hi": "Hindi",
	"tr": "Turkish",
	"uk": "Ukrainian",
	"pl": "Polish",
}

func buildSystemPrompt(lang string) string {
	langInstruction := ""
	if lang != "" && lang != "en" {
		name, ok := langNames[lang]
		if !ok {
			name = lang
		}
		langInstruction = fmt.Sprintf("\n- Respond entirely in %s.", name)
	}

	return fmt.Sprintf(`You are a tarot reader providing neutral, reflective interpretations.

Rules:
- Be maximally neutral and balanced.
- Never provide medical, legal, or financial advice.
- Never predict specific outcomes or disasters.
- Never command actions or diagnose conditions.
- Offer balanced possibilities and reflective questions.
- If a question is provided, incorporate it but never guarantee outcomes.%s

Respond with ONLY a JSON object (no markdown, no code fences, no extra text) matching this exact schema:
{
  "text": "<your interpretation>",
  "style": "neutral",
  "disclaimer": "For reflection/entertainment; not medical/legal/financial advice."
}`, langInstruction)
}

func buildUserPrompt(in ports.InterpretInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Deck: %s\nSpread: %s\n\nCards drawn:\n", in.DeckID, in.Spread)

	for _, card := range in.Cards {
		fmt.Fprintf(&b, "  Position %d: %s (%s)\n", card.Position, card.Name, card.Orientation)
		fmt.Fprintf(&b, "    Keywords: %s\n", strings.Join(card.Keywords, ", "))
		fmt.Fprintf(&b, "    Meaning: %s\n", card.Short)
	}

	if in.Question != "" {
		fmt.Fprintf(&b, "\nThe querent asks: %q\n", in.Question)
	}

	b.WriteString("\nProvide a cohesive interpretation as a single JSON object.")
	return b.String()
}

func retryPrompt(badJSON string) string {
	return fmt.Sprintf(`Your previous response was not valid JSON. Here is what you returned:
%s

Return ONLY the corrected JSON object matching this schema (no markdown, no code fences):
{
  "text": "<your interpretation>",
  "style": "neutral",
  "disclaimer": "For reflection/entertainment; not medical/legal/financial advice."
}`, badJSON)
}
