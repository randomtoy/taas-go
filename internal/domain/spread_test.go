package domain_test

import (
	"testing"

	"github.com/randomtoy/taas-go/internal/domain"
)

// deterministicRNG returns values from a pre-set sequence.
type deterministicRNG struct {
	values []int
	idx    int
}

func (r *deterministicRNG) Intn(n int) int {
	v := r.values[r.idx%len(r.values)] % n
	r.idx++
	return v
}

func testDeck(n int) domain.Deck {
	cards := make([]domain.Card, n)
	for i := range n {
		cards[i] = domain.Card{
			ID:       "card_" + string(rune('a'+i)),
			Name:     "Card " + string(rune('A'+i)),
			Keywords: []string{"kw1", "kw2"},
			Short:    "Short description.",
		}
	}
	return domain.Deck{ID: "test", Name: "Test Deck", Cards: cards}
}

func TestGenerateSpread_ThreeUniqueCards(t *testing.T) {
	deck := testDeck(22)
	// RNG sequence: shuffle uses values, then orientation uses values.
	rng := &deterministicRNG{values: []int{
		// Shuffle (21 swaps): all zeros keeps original order
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		// Orientation for 3 cards: 0=upright, 1=reversed, 0=upright
		0, 1, 0,
	}}

	spread, err := domain.GenerateSpread(deck, 3, domain.SpreadThreeCard, rng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(spread.Cards) != 3 {
		t.Fatalf("expected 3 cards, got %d", len(spread.Cards))
	}

	// Check uniqueness.
	seen := make(map[string]bool)
	for _, c := range spread.Cards {
		if seen[c.ID] {
			t.Errorf("duplicate card ID: %s", c.ID)
		}
		seen[c.ID] = true
	}
}

func TestGenerateSpread_Positions(t *testing.T) {
	deck := testDeck(10)
	rng := &deterministicRNG{values: []int{
		0, 0, 0, 0, 0, 0, 0, 0, 0, // shuffle (9 swaps for 10 cards)
		0, 0, 0, // orientation
	}}

	spread, err := domain.GenerateSpread(deck, 3, domain.SpreadGeneric, rng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, c := range spread.Cards {
		if c.Position != i+1 {
			t.Errorf("card %d: expected position %d, got %d", i, i+1, c.Position)
		}
	}
}

func TestGenerateSpread_Orientation(t *testing.T) {
	deck := testDeck(5)
	rng := &deterministicRNG{values: []int{
		0, 0, 0, 0, // shuffle (4 swaps for 5 cards)
		0, 1, 0, // orientation: upright, reversed, upright
	}}

	spread, err := domain.GenerateSpread(deck, 3, domain.SpreadThreeCard, rng)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []domain.Orientation{domain.Upright, domain.Reversed, domain.Upright}
	for i, c := range spread.Cards {
		if c.Orientation != expected[i] {
			t.Errorf("card %d: expected %s, got %s", i, expected[i], c.Orientation)
		}
	}
}

func TestGenerateSpread_InvalidN(t *testing.T) {
	deck := testDeck(5)
	rng := &deterministicRNG{values: []int{0}}

	for _, n := range []int{0, -1, 11} {
		_, err := domain.GenerateSpread(deck, n, domain.SpreadGeneric, rng)
		if err != domain.ErrInvalidN {
			t.Errorf("n=%d: expected ErrInvalidN, got %v", n, err)
		}
	}
}

func TestGenerateSpread_NExceedsDeck(t *testing.T) {
	deck := testDeck(2)
	rng := &deterministicRNG{values: []int{0}}

	_, err := domain.GenerateSpread(deck, 5, domain.SpreadGeneric, rng)
	if err != domain.ErrNExceedsDeck {
		t.Errorf("expected ErrNExceedsDeck, got %v", err)
	}
}
