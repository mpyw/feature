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
// # Named Keys for Debugging
//
// You can create named keys to help with debugging:
//
//	var MyFeature = feature.NewNamedBool("my-feature")
//	fmt.Println(MyFeature) // Output: my-feature
//
// Anonymous keys (without a name) automatically include call site information:
//
//	var AnonFeature = feature.NewBool()
//	fmt.Println(AnonFeature) // Output: anonymous(/path/to/file.go:42)@0x14000010098
//
// # Value-Based Feature Flags
//
// You can also use feature flags with arbitrary types:
//
//	var MaxItemsKey = feature.New[int]()
//	ctx = MaxItemsKey.WithValue(ctx, 100)
//	limit := MaxItemsKey.Get(ctx) // Returns 100
//
// # Inspecting Values
//
// Use Inspect to retrieve both the value and whether it was set in one call:
//
//	var MaxItems = feature.NewNamed[int]("max-items")
//	inspection := MaxItems.Inspect(ctx)
//	fmt.Println(inspection)         // Output: "max-items: 100" or "max-items: <not set>"
//	fmt.Println(inspection.IsSet()) // Output: true or false
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
	"runtime"
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

	// Inspect retrieves the value from the context and returns an Inspection
	// that provides convenient methods for working with the result.
	Inspect(ctx context.Context) Inspection[V]

	fmt.Stringer

	fmt.GoStringer

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

	// InspectBool retrieves the value from the context and returns a BoolInspection
	// that provides convenience methods for working with boolean feature flags.
	InspectBool(ctx context.Context) BoolInspection
}

// Option is a function that configures the behavior of a feature flag key.
type Option func(*options)

// options configures the behavior of a feature flag key.
type options struct {
	name string

	// internal use only - tracks the caller depth for name fallback
	depth int
}

// WithName returns an option that sets a debug name for the key.
// This name is included in the String() output for easier debugging.
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

// appendCallerDepthIncr appends an option that increments the caller depth for name fallback.
// This is used internally to ensure correct caller depth when deriving names from call sites.
func appendCallerDepthIncr(opts []Option) []Option {
	return append(opts, func(o *options) {
		o.depth++
	})
}

// defaultOptions returns a new options with default values.
func defaultOptions() *options {
	return &options{
		name:  "",
		depth: 0,
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

func computeKeyName(ident *opaque, name string, depth int) string {
	// Resolve the base name (handle anonymous keys)
	if name == "" {
		// Default fallback
		name = fmt.Sprintf("anonymous@%p", ident)
		// Enhance with call site info if available.
		// depth is the number of stack frames added by wrapper functions.
		// Each exported function (New, NewBool, NewNamed, NewNamedBool) calls appendCallerDepthIncr.
		// The call stack is: runtime.Caller -> computeKeyName -> New -> [wrappers...] -> user code
		// Base offset is 1 (computeKeyName itself), plus depth for wrapper functions.
		if _, file, line, ok := runtime.Caller(1 + depth); ok {
			name = fmt.Sprintf("anonymous(%s:%d)@%p", file, line, ident)
		}
	}

	return name
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
	options = appendCallerDepthIncr(options)

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
	options = appendCallerDepthIncr(options)

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
	options = appendCallerDepthIncr(options)
	opts := optionsFrom(options)
	ident := new(opaque)

	return key[V]{
		name:  computeKeyName(ident, opts.name, opts.depth),
		ident: ident,
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
	options = appendCallerDepthIncr(options)

	return New[V](append([]Option{WithName(name)}, options...)...)
}

// key is the internal implementation of Key[V].
type key[V any] struct {
	name  string
	ident *opaque
}

// boolKey is the internal implementation of BoolKey.
type boolKey struct {
	key[bool]
}

// String returns the debug name of the key.
// This implements fmt.Stringer.
func (k key[V]) String() string {
	return k.name
}

// GoString returns a Go syntax representation of the key.
// This implements fmt.GoStringer.
func (k key[V]) GoString() string {
	return fmt.Sprintf("feature.Key[%T]{name: %q}", *new(V), k.name)
}

// Inspect retrieves the value from the context and returns an Inspection.
func (k key[V]) Inspect(ctx context.Context) Inspection[V] {
	val, ok := k.TryGet(ctx)

	return Inspection[V]{
		Key:   k,
		Value: val,
		Ok:    ok,
	}
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
	return k.Inspect(ctx).Get()
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
	return k.Inspect(ctx).GetOrDefault(defaultValue)
}

// MustGet retrieves the value associated with this key from the context.
// If the key is not set, it panics with a descriptive error message.
func (k key[V]) MustGet(ctx context.Context) V {
	return k.Inspect(ctx).MustGet()
}

// IsSet returns true if this key has been set in the context.
func (k key[V]) IsSet(ctx context.Context) bool {
	return k.Inspect(ctx).IsSet()
}

// IsNotSet returns true if this key has not been set in the context.
func (k key[V]) IsNotSet(ctx context.Context) bool {
	return k.Inspect(ctx).IsNotSet()
}

// InspectBool retrieves the value from the context and returns a BoolInspection.
func (k boolKey) InspectBool(ctx context.Context) BoolInspection {
	return BoolInspection{Inspection: k.Inspect(ctx)}
}

// Enabled returns true if the feature flag is set to true in the context.
func (k boolKey) Enabled(ctx context.Context) bool {
	return k.InspectBool(ctx).Enabled()
}

// Disabled returns true if the feature flag is either not set or set to false.
func (k boolKey) Disabled(ctx context.Context) bool {
	return k.InspectBool(ctx).Disabled()
}

// ExplicitlyDisabled returns true if the feature flag is explicitly set to false.
func (k boolKey) ExplicitlyDisabled(ctx context.Context) bool {
	return k.InspectBool(ctx).ExplicitlyDisabled()
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
