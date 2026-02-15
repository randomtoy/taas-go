package app_test

import (
	"context"
	"testing"

	"github.com/randomtoy/taas-go/internal/app"
	"github.com/randomtoy/taas-go/internal/domain"
	"github.com/randomtoy/taas-go/internal/ports"
)

type mockDeckStore struct {
	deck domain.Deck
	err  error
}

func (m *mockDeckStore) GetDeck(_ context.Context, _ string) (domain.Deck, error) {
	return m.deck, m.err
}

type mockInterpreter struct {
	out ports.InterpretOutput
	err error
}

func (m *mockInterpreter) Interpret(_ context.Context, _ ports.InterpretInput) (ports.InterpretOutput, error) {
	return m.out, m.err
}

type fixedRNG struct{ val int }

func (r fixedRNG) Intn(n int) int { return r.val % n }

func testDeck() domain.Deck {
	cards := make([]domain.Card, 22)
	for i := range 22 {
		cards[i] = domain.Card{
			ID:       "card_" + string(rune('a'+i)),
			Name:     "Card " + string(rune('A'+i)),
			Keywords: []string{"kw1"},
			Short:    "Short.",
		}
	}
	return domain.Deck{ID: "major_arcana", Name: "Major Arcana", Cards: cards}
}

func TestReadSpread_Success(t *testing.T) {
	ds := &mockDeckStore{deck: testDeck()}
	interp := &mockInterpreter{
		out: ports.InterpretOutput{
			Text:       "An insightful interpretation.",
			Style:      "neutral",
			Disclaimer: "For reflection only.",
		},
	}
	svc := app.NewTarotService(ds, interp, fixedRNG{val: 0}, "test-model")

	resp, err := svc.ReadSpread(context.Background(), app.ReadSpreadRequest{
		Question:   "Will it rain?",
		NumCards:   3,
		DeckID:     "major_arcana",
		SpreadType: "three_card",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Cards) != 3 {
		t.Fatalf("expected 3 cards, got %d", len(resp.Cards))
	}
	if resp.Interpretation.Text != "An insightful interpretation." {
		t.Errorf("unexpected interpretation text: %s", resp.Interpretation.Text)
	}
	if resp.Model != "test-model" {
		t.Errorf("unexpected model: %s", resp.Model)
	}
}

func TestReadSpread_DeckNotFound(t *testing.T) {
	ds := &mockDeckStore{err: domain.ErrDeckNotFound}
	interp := &mockInterpreter{}
	svc := app.NewTarotService(ds, interp, fixedRNG{val: 0}, "test-model")

	_, err := svc.ReadSpread(context.Background(), app.ReadSpreadRequest{
		NumCards:   3,
		DeckID:     "nonexistent",
		SpreadType: "generic",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestReadSpread_LLMFailure(t *testing.T) {
	ds := &mockDeckStore{deck: testDeck()}
	interp := &mockInterpreter{err: domain.ErrUpstreamLLM}
	svc := app.NewTarotService(ds, interp, fixedRNG{val: 0}, "test-model")

	_, err := svc.ReadSpread(context.Background(), app.ReadSpreadRequest{
		NumCards:   3,
		DeckID:     "major_arcana",
		SpreadType: "three_card",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
