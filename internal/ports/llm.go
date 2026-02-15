package ports

import "context"

// InterpretInput holds everything the LLM needs to generate an interpretation.
type InterpretInput struct {
	DeckID   string
	Spread   string
	Question string
	Cards    []CardInput
	Lang     string // BCP 47 language code, e.g. "en", "ru", "es"
}

// CardInput is a simplified card representation for the LLM prompt.
type CardInput struct {
	Name        string
	Position    int
	Orientation string
	Keywords    []string
	Short       string
}

// InterpretOutput is the structured interpretation returned by the LLM.
type InterpretOutput struct {
	Text       string `json:"text"`
	Style      string `json:"style"`
	Disclaimer string `json:"disclaimer"`
	Model      string `json:"-"` // set by adapter, not from LLM JSON
}

// Interpreter generates a tarot interpretation via an LLM.
type Interpreter interface {
	Interpret(ctx context.Context, in InterpretInput) (InterpretOutput, error)
}
