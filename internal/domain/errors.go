package domain

import "errors"

var (
	ErrInvalidN      = errors.New("n must be between 1 and 10")
	ErrNExceedsDeck  = errors.New("n exceeds number of cards in deck")
	ErrDeckNotFound  = errors.New("deck not found")
	ErrUpstreamLLM   = errors.New("upstream LLM failure")
	ErrInvalidLLMJSON = errors.New("LLM returned invalid JSON after retry")
)
