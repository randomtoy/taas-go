package decks

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/randomtoy/taas-go/internal/domain"
)

//go:embed data/*.json
var deckFS embed.FS

// registry maps deck IDs to their JSON filenames inside data/.
var registry = map[string]string{
	"major_arcana": "data/major_arcana.json",
}

// EmbeddedStore loads decks from embedded JSON files.
type EmbeddedStore struct {
	once  sync.Once
	decks map[string]domain.Deck
	err   error
}

func NewEmbeddedStore() *EmbeddedStore {
	return &EmbeddedStore{}
}

func (s *EmbeddedStore) init() {
	s.decks = make(map[string]domain.Deck, len(registry))
	for id, filename := range registry {
		raw, err := deckFS.ReadFile(filename)
		if err != nil {
			s.err = fmt.Errorf("read embedded deck %s: %w", id, err)
			return
		}
		var cards []domain.Card
		if err := json.Unmarshal(raw, &cards); err != nil {
			s.err = fmt.Errorf("parse embedded deck %s: %w", id, err)
			return
		}
		s.decks[id] = domain.Deck{
			ID:    id,
			Name:  id,
			Cards: cards,
		}
	}
}

func (s *EmbeddedStore) GetDeck(_ context.Context, deckID string) (domain.Deck, error) {
	s.once.Do(s.init)
	if s.err != nil {
		return domain.Deck{}, s.err
	}
	deck, ok := s.decks[deckID]
	if !ok {
		return domain.Deck{}, domain.ErrDeckNotFound
	}
	return deck, nil
}
