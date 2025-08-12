package utils

import (
	"testing"
)

func TestAssertInvariant_ValidCondition(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AssertInvariant should not panic when condition is true, but got panic: %v", r)
		}
	}()

	AssertInvariant(true, "this should not panic")
}

func TestAssertInvariant_InvalidCondition(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("AssertInvariant should panic when condition is false")
			return
		}

		expectedMessage := "invariant violated - test message"
		if r != expectedMessage {
			t.Errorf("Expected panic message '%s', got '%v'", expectedMessage, r)
		}
	}()

	AssertInvariant(false, "test message")
}

func TestAssertInvariant_EmptyMessage(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("AssertInvariant should panic when condition is false, even with empty message")
			return
		}

		expectedMessage := "invariant violated - "
		if r != expectedMessage {
			t.Errorf("Expected panic message '%s', got '%v'", expectedMessage, r)
		}
	}()

	AssertInvariant(false, "")
}

func TestAssertInvariant_ComplexCondition(t *testing.T) {
	value := 42

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("AssertInvariant should not panic when complex condition is true, but got panic: %v", r)
		}
	}()

	AssertInvariant(value > 0 && value < 100, "value should be in range")
}

func TestAssertInvariant_ComplexConditionFails(t *testing.T) {
	value := -5

	defer func() {
		r := recover()
		if r == nil {
			t.Error("AssertInvariant should panic when complex condition is false")
			return
		}

		expectedMessage := "invariant violated - value must be positive"
		if r != expectedMessage {
			t.Errorf("Expected panic message '%s', got '%v'", expectedMessage, r)
		}
	}()

	AssertInvariant(value > 0, "value must be positive")
}
