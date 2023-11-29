package utils

// Result is the result of a function. useful for chan.
type Result[T any] struct {
	Err error
	Val T
}
