package model_test

import (
	"testing"

	"task282-ednadiag/internal/model"
)

func TestChainStatusConstants(t *testing.T) {
	if model.ChainReceiving == "" || model.ChainSealed == "" {
		t.Fatal("chain status constants must be non-empty")
	}
}

func TestIsNotFound(t *testing.T) {
	if !model.IsNotFound(model.ErrNotFound) {
		t.Fatal("ErrNotFound should be not found")
	}
}
