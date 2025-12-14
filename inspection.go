package feature

import "fmt"

// Inspection holds the result of inspecting a key's value in a context.
// It captures both the key, its value, and whether the value was set.
// This type provides convenient methods for working with the inspection result
// without needing to pass the context again.
type Inspection[V any] struct {
	// Key is the key that was inspected.
	Key Key[V]
	// Value is the value retrieved from the context.
	// If Ok is false, this will be the zero value of type V.
	Value V
	// Ok indicates whether the key was set in the context.
	Ok bool
}

// Get returns the value from the inspection.
// If the key was not set, it returns the zero value of type V.
func (i Inspection[V]) Get() V {
	return i.Value
}

// TryGet returns the value and whether it was set.
func (i Inspection[V]) TryGet() (V, bool) {
	return i.Value, i.Ok
}

// GetOrDefault returns the value if set, otherwise returns the provided default.
func (i Inspection[V]) GetOrDefault(defaultValue V) V {
	if i.Ok {
		return i.Value
	}

	return defaultValue
}

// MustGet returns the value if set, otherwise panics.
func (i Inspection[V]) MustGet() V {
	if !i.Ok {
		panic(fmt.Sprintf("key %s is not set in context", i.Key.String()))
	}

	return i.Value
}

// IsSet returns true if the key was set in the context.
func (i Inspection[V]) IsSet() bool {
	return i.Ok
}

// IsNotSet returns true if the key was not set in the context.
func (i Inspection[V]) IsNotSet() bool {
	return !i.Ok
}

// String returns a string representation combining the key name and its value.
// Format: "<key-name>: <value>" or "<key-name>: <not set>".
// This implements fmt.Stringer.
func (i Inspection[V]) String() string {
	if !i.Ok {
		return i.Key.String() + ": <not set>"
	}

	return fmt.Sprintf("%s: %v", i.Key.String(), i.Value)
}

// BoolInspection is a specialized Inspection for boolean feature flags.
// It provides convenience methods for working with boolean values.
type BoolInspection struct {
	Inspection[bool]
}

// Enabled returns true if the feature flag is set to true.
// If the key was not set, it returns false (the zero value).
func (i BoolInspection) Enabled() bool {
	return i.Value
}

// Disabled returns true if the feature flag is either not set or set to false.
func (i BoolInspection) Disabled() bool {
	return !i.Enabled()
}

// ExplicitlyDisabled returns true if the feature flag is explicitly set to false.
// It returns false if the key was not set (distinguishing from Disabled).
func (i BoolInspection) ExplicitlyDisabled() bool {
	return i.Ok && !i.Value
}

// String returns a string representation combining the key name and its value.
// Delegates to the embedded Inspection.String().
// This implements fmt.Stringer.
func (i BoolInspection) String() string {
	return i.Inspection.String()
}

