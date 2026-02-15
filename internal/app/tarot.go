package app

import (
	"context"
	"fmt"
	"time"

	"github.com/randomtoy/taas-go/internal/domain"
	"github.com/randomtoy/taas-go/internal/ports"
)

// ReadSpreadRequest is the application-level input (no HTTP types).
type ReadSpreadRequest struct {
	Question   string
	NumCards   int
	DeckID     string
	SpreadType string
}

// ReadSpreadResponse is the application-level output.
type ReadSpreadResponse struct {
	SpreadType     domain.SpreadType
	DeckID         string
	Cards          []domain.DrawnCard
	Interpretation ports.InterpretOutput
	Model          string
	LatencyMS      int64
}

// TarotService orchestrates spread generation and LLM interpretation.
type TarotService struct {
	deckStore   ports.DeckStore
	interpreter ports.Interpreter
	rng         domain.RNG
	model       string
}

func NewTarotService(ds ports.DeckStore, interp ports.Interpreter, rng domain.RNG, model string) *TarotService {
	return &TarotService{
		deckStore:   ds,
		interpreter: interp,
		rng:         rng,
		model:       model,
	}
}

func (s *TarotService) ReadSpread(ctx context.Context, req ReadSpreadRequest) (ReadSpreadResponse, error) {
	deck, err := s.deckStore.GetDeck(ctx, req.DeckID)
	if err != nil {
		return ReadSpreadResponse{}, fmt.Errorf("get deck: %w", err)
	}

	st := resolveSpreadType(req.SpreadType, req.NumCards)

	spread, err := domain.GenerateSpread(deck, req.NumCards, st, s.rng)
	if err != nil {
		return ReadSpreadResponse{}, fmt.Errorf("generate spread: %w", err)
	}

	llmInput := ports.InterpretInput{
		DeckID:   req.DeckID,
		Spread:   string(st),
		Question: req.Question,
		Cards:    toCardInputs(spread.Cards),
	}

	start := time.Now()
	interpretation, err := s.interpreter.Interpret(ctx, llmInput)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return ReadSpreadResponse{}, fmt.Errorf("interpret: %w", err)
	}

	return ReadSpreadResponse{
		SpreadType:     st,
		DeckID:         req.DeckID,
		Cards:          spread.Cards,
		Interpretation: interpretation,
		Model:          interpretationModel(interpretation.Model, s.model),
		LatencyMS:      latency,
	}, nil
}

func resolveSpreadType(raw string, n int) domain.SpreadType {
	switch raw {
	case "three_card":
		return domain.SpreadThreeCard
	case "generic", "":
		if n == 3 {
			return domain.SpreadThreeCard
		}
		return domain.SpreadGeneric
	default:
		return domain.SpreadType(raw)
	}
}

func interpretationModel(fromLLM, fallback string) string {
	if fromLLM != "" {
		return fromLLM
	}
	return fallback
}

func toCardInputs(cards []domain.DrawnCard) []ports.CardInput {
	out := make([]ports.CardInput, len(cards))
	for i, c := range cards {
		out[i] = ports.CardInput{
			Name:        c.Name,
			Position:    c.Position,
			Orientation: string(c.Orientation),
			Keywords:    c.Keywords,
			Short:       c.Short,
		}
	}
	return out
}
