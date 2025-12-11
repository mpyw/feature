// Package feature provides a type-safe, collision-free feature flag implementation using Go's context.
//
// This package allows you to define feature flags as strongly-typed keys that can be stored
// in and retrieved from a context.Context. Each key is guaranteed to be unique based on its
// pointer identity, preventing accidental collisions even when using the same type.
//
// # Basic Usage
//
// Define a boolean feature flag:
//
//	var MyFeature = feature.NewBool()
//
//	func Handler(ctx context.Context) {
//	    if MyFeature.Enabled(ctx) {
//	        // New feature implementation
//	    } else {
//	        // Legacy implementation
//	    }
//	}
//
// Enable the feature flag:
//
//	ctx = MyFeature.WithEnabled(ctx)
//
// # WithName Keys for Debugging
//
// You can create named keys to help with debugging:
//
//	var MyFeature = feature.NewNamedBool("my-feature")
//	fmt.Println(MyFeature) // Output: my-feature
//
// # Value-Based Feature Flags
//
// You can also use feature flags with arbitrary types:
//
//	var MaxItemsKey = feature.New[int]()
//	ctx = MaxItemsKey.WithValue(ctx, 100)
//	limit := MaxItemsKey.Get(ctx) // Returns 100
//
// # Key Properties
//
//   - Type-safe: Uses generics to ensure type safety at compile time
//   - Collision-free: Each key is unique based on its pointer identity
//   - Context-safe: Follows Go's context immutability guarantees
//   - Zero dependencies: No external dependencies required
package feature

import (
	"context"
	"fmt"
)

// Key is a type-safe accessor for feature flags stored in context.Context.
//
// Each Key instance is uniquely identified by its pointer address, preventing collisions
// even when multiple keys use the same value type. Type parameter V specifies the type
// of value stored by this key.
//
// Keys are safe for concurrent use and follow Go's context immutability guarantees.
type Key[V any] interface {
	// WithValue returns a new context with the given value associated with this key.
	// The original context is not modified.
	WithValue(ctx context.Context, value V) context.Context

	// Get retrieves the value associated with this key from the context.
	// If the key is not set in the context, it returns the zero value of type V.
	Get(ctx context.Context) V

	// TryGet attempts to retrieve the value associated with this key from the context.
	// It returns the value and a boolean indicating whether the key was set in the context.
	// If the key is not set, it returns the zero value of type V and false.
	TryGet(ctx context.Context) (V, bool)

	// GetOrDefault retrieves the value associated with this key from the context.
	// If the key is not set, it returns the provided default value.
	GetOrDefault(ctx context.Context, defaultValue V) V

	// MustGet retrieves the value associated with this key from the context.
	// If the key is not set, it panics with a descriptive error message.
	MustGet(ctx context.Context) V

	// IsSet returns true if this key has been set in the context.
	// It returns false if the key is not present, regardless of what the zero value would be.
	IsSet(ctx context.Context) bool

	// IsNotSet returns true if this key has not been set in the context.
	// It is equivalent to !IsSet(ctx).
	IsNotSet(ctx context.Context) bool

	// DebugValue returns a string representation combining the key name and its value from the context.
	// This is useful for debugging and logging purposes.
	// Format: "<key-name>: <value>" or "<key-name>: <not set>".
	DebugValue(ctx context.Context) string

	fmt.Stringer

	// downcast is an internal method used to retrieve the underlying key implementation.
	// also used for sealing the interface.
	downcast() key[V]
}

// BoolKey is a specialized Key for boolean feature flags.
//
// It embeds Key[bool] and provides convenience methods for common boolean operations,
// making it more ergonomic to work with feature flags that represent on/off states.
type BoolKey interface {
	Key[bool]

	// Enabled returns true if the feature flag is set to true in the context.
	// If the key is not set in the context, it returns false (the zero value).
	Enabled(ctx context.Context) bool

	// Disabled returns true if the feature flag is either not set or set to false.
	// This is equivalent to !Enabled(ctx).
	Disabled(ctx context.Context) bool

	// ExplicitlyDisabled returns true if the feature flag is explicitly set to false.
	// It returns false if the key is not set in the context (distinguishing from Disabled).
	ExplicitlyDisabled(ctx context.Context) bool

	// WithEnabled returns a new context with this feature flag enabled (set to true).
	// The original context is not modified.
	WithEnabled(ctx context.Context) context.Context

	// WithDisabled returns a new context with this feature flag disabled (set to false).
	// The original context is not modified.
	WithDisabled(ctx context.Context) context.Context
}

// StringerFunc is a function that formats a key name as a string.
// It receives a resolved key name (never empty - anonymous keys are already
// formatted as "anonymous@<address>") and returns the final string representation.
// The default implementation returns the name as-is.
type StringerFunc func(name string) string

// Option is a function that configures the behavior of a feature flag key.
type Option func(*options)

// options configures the behavior of a feature flag key.
type options struct {
	name     string
	stringer StringerFunc
}

// WithName returns an option that sets a debug name for the key.
// This name is included in the String() output and used in DebugValue() for easier debugging.
//
// Example:
//
//	var MyKey = feature.New[int](feature.WithName("max-retries"))
//	fmt.Println(MyKey.String()) // Output: max-retries
func WithName(name string) Option {
	return func(o *options) {
		o.name = name
	}
}

// WithStringer returns an option that sets a custom string formatter for the key name.
// The formatter function receives a resolved key name (never empty) and returns
// the final string representation.
//
// If not provided, the default formatter returns the name as-is.
//
// Example:
//
//	customFormatter := func(name string) string {
//	    return fmt.Sprintf("[%s]", name)
//	}
//	var MyKey = feature.New[int](feature.WithStringer(customFormatter))
func WithStringer(f StringerFunc) Option {
	return func(o *options) {
		o.stringer = f
	}
}

// defaultStringer is the default string formatter for keys.
// It returns the name as-is (identity function).
func defaultStringer(name string) string {
	return name
}

// defaultOptions returns a new options with default values.
func defaultOptions() *options {
	return &options{
		name:     "",
		stringer: defaultStringer,
	}
}

// optionsFrom applies the given option functions to create a configured options.
func optionsFrom(opts []Option) *options {
	o := defaultOptions()
	for _, optFn := range opts {
		optFn(o)
	}

	return o
}

// NewBool creates a new boolean feature flag key.
//
// Each call to NewBool creates a unique key based on pointer identity, preventing collisions.
// The key can be configured with optional configuration functions.
//
// Example:
//
//	var EnableNewUI = feature.NewBool()
//
//	func ShowUI(ctx context.Context) {
//	    if EnableNewUI.Enabled(ctx) {
//	        // Show new UI
//	    }
//	}
func NewBool(options ...Option) BoolKey {
	return boolKey{key: New[bool](options...).downcast()}
}

// NewNamedBool creates a new boolean feature flag key with a debug name.
//
// This is a convenience function equivalent to calling NewBool(feature.WithName(name), ...).
// The name is included in the String() output for easier debugging.
//
// Example:
//
//	var EnableNewUI = feature.NewNamedBool("new-ui")
//	fmt.Println(EnableNewUI) // Output: new-ui
func NewNamedBool(name string, options ...Option) BoolKey {
	return NewBool(append([]Option{WithName(name)}, options...)...)
}

// New creates a new feature flag key for values of type V.
//
// Each call to New creates a unique key based on pointer identity, preventing collisions.
// The key can be configured with optional configuration functions.
//
// Example:
//
//	var MaxRetries = feature.New[int]()
//	ctx = MaxRetries.WithValue(ctx, 5)
//	retries := MaxRetries.Get(ctx) // Returns 5
func New[V any](options ...Option) Key[V] {
	opts := optionsFrom(options)

	return key[V]{
		name:     opts.name,
		stringer: opts.stringer,
		ident:    new(opaque),
	}
}

// NewNamed creates a new feature flag key for values of type V with a debug name.
//
// This is a convenience function equivalent to calling New[V](feature.WithName(name), ...).
// The name is included in the String() output for easier debugging.
//
// Example:
//
//	var MaxRetries = feature.NewNamed[int]("max-retries")
//	fmt.Println(MaxRetries) // Output: max-retries
func NewNamed[V any](name string, options ...Option) Key[V] {
	return New[V](append([]Option{WithName(name)}, options...)...)
}

// key is the internal implementation of Key[V].
type key[V any] struct {
	name     string
	stringer StringerFunc
	ident    *opaque
}

// boolKey is the internal implementation of BoolKey.
type boolKey struct {
	key[bool]
}

// String returns a string representation of the key name.
// The format can be customized via the WithStringer option.
// By default, it returns the debug name if provided, or "anonymous@<address>" otherwise.
func (k key[V]) String() string {
	// Resolve the base name (handle anonymous keys)
	name := k.name
	if name == "" {
		name = fmt.Sprintf("anonymous@%p", k.ident)
	}

	// Apply custom stringer if provided
	stringer := k.stringer
	if stringer == nil {
		stringer = defaultStringer
	}

	return stringer(name)
}

// DebugValue returns a string representation combining the key name and its value from the context.
// This is useful for debugging and logging purposes.
// Format: "<key-name>: <value>" or "<key-name>: <not set>".
func (k key[V]) DebugValue(ctx context.Context) string {
	keyName := k.String()
	val, ok := k.TryGet(ctx)

	if !ok {
		return keyName + ": <not set>"
	}

	return fmt.Sprintf("%s: %v", keyName, val)
}

func (k key[V]) downcast() key[V] {
	return k
}

// WithValue returns a new context with the given value associated with this key.
func (k key[V]) WithValue(ctx context.Context, value V) context.Context {
	return context.WithValue(ctx, k.ident, value)
}

// Get retrieves the value associated with this key from the context.
// If the key is not set in the context, it returns the zero value of type V.
func (k key[V]) Get(ctx context.Context) V {
	val, _ := k.TryGet(ctx)

	return val
}

// TryGet attempts to retrieve the value associated with this key from the context.
// It returns the value and a boolean indicating whether the key was set in the context.
func (k key[V]) TryGet(ctx context.Context) (V, bool) {
	val, ok := ctx.Value(k.ident).(V)

	return val, ok
}

// GetOrDefault retrieves the value associated with this key from the context.
// If the key is not set, it returns the provided default value.
func (k key[V]) GetOrDefault(ctx context.Context, defaultValue V) V {
	if val, ok := k.TryGet(ctx); ok {
		return val
	}

	return defaultValue
}

// MustGet retrieves the value associated with this key from the context.
// If the key is not set, it panics with a descriptive error message.
func (k key[V]) MustGet(ctx context.Context) V {
	val, ok := k.TryGet(ctx)
	if !ok {
		panic(fmt.Sprintf("key %s is not set in context", k.String()))
	}

	return val
}

// IsSet returns true if this key has been set in the context.
func (k key[V]) IsSet(ctx context.Context) bool {
	_, ok := k.TryGet(ctx)

	return ok
}

// IsNotSet returns true if this key has not been set in the context.
func (k key[V]) IsNotSet(ctx context.Context) bool {
	return !k.IsSet(ctx)
}

// Enabled returns true if the feature flag is set to true in the context.
func (k boolKey) Enabled(ctx context.Context) bool {
	return k.Get(ctx)
}

// Disabled returns true if the feature flag is either not set or set to false.
func (k boolKey) Disabled(ctx context.Context) bool {
	return !k.Enabled(ctx)
}

// ExplicitlyDisabled returns true if the feature flag is explicitly set to false.
func (k boolKey) ExplicitlyDisabled(ctx context.Context) bool {
	val, ok := k.TryGet(ctx)

	return ok && !val
}

// WithEnabled returns a new context with this feature flag enabled (set to true).
func (k boolKey) WithEnabled(ctx context.Context) context.Context {
	return k.WithValue(ctx, true)
}

// WithDisabled returns a new context with this feature flag disabled (set to false).
func (k boolKey) WithDisabled(ctx context.Context) context.Context {
	return k.WithValue(ctx, false)
}

type opaque struct {
	// Include a byte field to prevent the compiler from optimizing away
	// all instances to the same memory address.
	_ byte
}
