package domain

// GenerateSpread draws n unique cards from deck using the provided RNG.
// Positions are 1-based. Orientation is 50/50 upright/reversed.
func GenerateSpread(deck Deck, n int, spreadType SpreadType, rng RNG) (Spread, error) {
	if n < 1 || n > 10 {
		return Spread{}, ErrInvalidN
	}
	if n > len(deck.Cards) {
		return Spread{}, ErrNExceedsDeck
	}

	// Fisher-Yates partial shuffle: only need first n elements.
	indices := make([]int, len(deck.Cards))
	for i := range indices {
		indices[i] = i
	}
	for i := len(indices) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		indices[i], indices[j] = indices[j], indices[i]
	}

	cards := make([]DrawnCard, n)
	for i := range n {
		orientation := Upright
		if rng.Intn(2) == 1 {
			orientation = Reversed
		}
		cards[i] = DrawnCard{
			Card:        deck.Cards[indices[i]],
			Position:    i + 1,
			Orientation: orientation,
		}
	}

	return Spread{
		Type:  spreadType,
		Cards: cards,
	}, nil
}
