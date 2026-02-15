package ports

import (
	"context"

	"github.com/randomtoy/taas-go/internal/domain"
)

// DeckStore provides access to tarot decks.
type DeckStore interface {
	GetDeck(ctx context.Context, deckID string) (domain.Deck, error)
}
