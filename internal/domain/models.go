package domain

// RNG abstracts random number generation for deterministic testing.
type RNG interface {
	// Intn returns a non-negative random int in [0, n).
	Intn(n int) int
}

// Orientation represents the orientation of a drawn tarot card.
type Orientation string

const (
	Upright  Orientation = "upright"
	Reversed Orientation = "reversed"
)

// Card represents a single tarot card in a deck.
type Card struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
	Short    string   `json:"short"`
}

// DrawnCard is a card that has been drawn as part of a spread.
type DrawnCard struct {
	Card
	Position    int         `json:"position"`
	Orientation Orientation `json:"orientation"`
}

// Deck is a collection of tarot cards.
type Deck struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Cards []Card `json:"cards"`
}

// SpreadType identifies the type of spread.
type SpreadType string

const (
	SpreadGeneric   SpreadType = "generic"
	SpreadThreeCard SpreadType = "three_card"
)

// Spread is the result of drawing cards from a deck.
type Spread struct {
	Type  SpreadType
	Cards []DrawnCard
}
