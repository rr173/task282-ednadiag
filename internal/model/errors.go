package model

import "errors"

// 领域错误。
var (
	ErrNotFound            = errors.New("entity not found")
	ErrInvalidInput        = errors.New("invalid input")
	ErrBatchMissing        = errors.New("referenced batch does not exist")
	ErrFeatureCodeInvalid  = errors.New("feature encoding invalid: taxon and marker must be non-empty and ascii")
	ErrChainCycle          = errors.New("chain cycle detected: step would create a dependency loop")
	ErrSealedImmutable     = errors.New("sealed entity is immutable")
	ErrDuplicateID         = errors.New("duplicate entity id")
	ErrIllegalTransition   = errors.New("illegal state transition")
	ErrEmptyChain          = errors.New("chain has no steps")
)

// IsNotFound 判断是否为未找到错误。
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
