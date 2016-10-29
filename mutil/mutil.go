package mutil

type BinaryChangeIntent int

const (
	IntentNone = iota
	IntentAdd
	IntentRemove
	IntentFlip
)
