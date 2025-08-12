package utils

func AssertInvariant(condition bool, message string) {
	if !condition {
		panic("invariant violated - " + message)
	}
}
