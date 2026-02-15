package http

import "github.com/randomtoy/taas-go/internal/domain"

// TarotResponse is the JSON shape returned by GET /v1/tarot.
type TarotResponse struct {
	Spread         string             `json:"spread"`
	Deck           string             `json:"deck"`
	Cards          []CardResponse     `json:"cards"`
	Interpretation InterpretationResp `json:"interpretation"`
	Meta           MetaResp           `json:"meta"`
}

type CardResponse struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Position    int             `json:"position"`
	Orientation domain.Orientation `json:"orientation"`
	Keywords    []string        `json:"keywords"`
	Short       string          `json:"short"`
}

type InterpretationResp struct {
	Style      string `json:"style"`
	Text       string `json:"text"`
	Disclaimer string `json:"disclaimer"`
}

type MetaResp struct {
	Model     string `json:"model"`
	RequestID string `json:"request_id"`
	LatencyMS int64  `json:"latency_ms"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
