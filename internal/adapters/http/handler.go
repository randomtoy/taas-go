package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/randomtoy/taas-go/internal/app"
	"github.com/randomtoy/taas-go/internal/domain"
)

type Handler struct {
	svc *app.TarotService
}

func NewHandler(svc *app.TarotService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(e *echo.Echo) {
	e.GET("/healthz", h.Healthz)
	e.GET("/v1/tarot", h.ReadTarot)
}

func (h *Handler) Healthz(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}

func (h *Handler) ReadTarot(c echo.Context) error {
	q := c.QueryParam("q")
	if len(q) > 500 {
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "q must be at most 500 characters"})
	}

	n := 3
	if raw := c.QueryParam("n"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 10 {
			return c.JSON(http.StatusBadRequest, ErrorResponse{Error: "n must be an integer between 1 and 10"})
		}
		n = parsed
	}

	deckID := c.QueryParam("deck")
	if deckID == "" {
		deckID = "major_arcana"
	}

	spread := c.QueryParam("spread")
	if spread == "" {
		spread = "generic"
	}

	req := app.ReadSpreadRequest{
		Question:   q,
		NumCards:   n,
		DeckID:     deckID,
		SpreadType: spread,
	}

	resp, err := h.svc.ReadSpread(c.Request().Context(), req)
	if err != nil {
		return mapError(c, err)
	}

	requestID, _ := c.Get("request_id").(string)

	return c.JSON(http.StatusOK, toResponse(resp, requestID))
}

func toResponse(r app.ReadSpreadResponse, requestID string) TarotResponse {
	cards := make([]CardResponse, len(r.Cards))
	for i, dc := range r.Cards {
		cards[i] = CardResponse{
			ID:          dc.ID,
			Name:        dc.Name,
			Position:    dc.Position,
			Orientation: dc.Orientation,
			Keywords:    dc.Keywords,
			Short:       dc.Short,
		}
	}
	return TarotResponse{
		Spread: string(r.SpreadType),
		Deck:   r.DeckID,
		Cards:  cards,
		Interpretation: InterpretationResp{
			Style:      r.Interpretation.Style,
			Text:       r.Interpretation.Text,
			Disclaimer: r.Interpretation.Disclaimer,
		},
		Meta: MetaResp{
			Model:     r.Model,
			RequestID: requestID,
			LatencyMS: r.LatencyMS,
		},
	}
}

func mapError(c echo.Context, err error) error {
	requestID, _ := c.Get("request_id").(string)

	switch {
	case errors.Is(err, domain.ErrDeckNotFound):
		return c.JSON(http.StatusNotFound, ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrInvalidN), errors.Is(err, domain.ErrNExceedsDeck):
		return c.JSON(http.StatusBadRequest, ErrorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrUpstreamLLM), errors.Is(err, domain.ErrInvalidLLMJSON):
		slog.Error("upstream LLM failure", "request_id", requestID, "error", err)
		return c.JSON(http.StatusBadGateway, ErrorResponse{Error: "upstream LLM failure"})
	default:
		slog.Error("internal error", "request_id", requestID, "error", err)
		return c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
	}
}
