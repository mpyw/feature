# feature

[![Go Reference](https://pkg.go.dev/badge/github.com/mpyw/feature.svg)](https://pkg.go.dev/github.com/mpyw/feature)
[![Test](https://github.com/mpyw/feature/actions/workflows/test.yaml/badge.svg)](https://github.com/mpyw/feature/actions/workflows/test.yaml)
[![Codecov](https://codecov.io/gh/mpyw/feature/graph/badge.svg?token=21VJVOVMY0)](https://codecov.io/gh/mpyw/feature)
[![Go Report Card](https://goreportcard.com/badge/github.com/mpyw/feature)](https://goreportcard.com/report/github.com/mpyw/feature)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A type-safe, collision-free feature flag implementation for Go using context.

## Features

- **Type-Safe**: Uses Go generics for compile-time type safety
- **Collision-Free**: Each key is uniquely identified by its pointer address
- **Context-Safe**: Follows Go's context immutability guarantees
- **Zero Dependencies**: No external dependencies required
- **Simple API**: Easy to use with minimal boilerplate

## Installation

```bash
go get github.com/mpyw/feature
```

Requires Go 1.21 or later.

## Quick Start

### Boolean Feature Flags

```go
package main

import (
    "context"
    "fmt"

    "github.com/mpyw/feature"
)

// Define a feature flag
var EnableNewUI = feature.NewNamedBool("new-ui")

func main() {
    ctx := context.Background()

    // Enable the feature
    ctx = EnableNewUI.WithEnabled(ctx)

    // Check if enabled
    if EnableNewUI.Enabled(ctx) {
        fmt.Println("New UI is enabled!")
    }
}
```

### Value-Based Feature Flags

```go
package main

import (
    "context"
    "fmt"

    "github.com/mpyw/feature"
)

// Define a feature flag with a custom type
var MaxRetries = feature.New[int]()

func main() {
    ctx := context.Background()

    // Set a value
    ctx = MaxRetries.WithValue(ctx, 5)

    // Retrieve the value
    retries := MaxRetries.Get(ctx)
    fmt.Printf("Max retries: %d\n", retries)
}
```

## API Overview

### Creating Keys

```go
// Boolean feature flag
var MyFeature = feature.NewBool()

// Boolean feature flag with a debug name
var MyNamedFeature = feature.NewNamedBool("my-feature")

// Feature flag with custom type
var MyValueKey = feature.New[string]()

// Feature flag with custom type and debug name
var MyNamedValueKey = feature.NewNamed[string]("my-key")
```

### Working with Boolean Flags

```go
// Enable a feature
ctx = MyFeature.WithEnabled(ctx)

// Disable a feature
ctx = MyFeature.WithDisabled(ctx)

// Check if enabled
if MyFeature.Enabled(ctx) {
    // Feature is enabled
}

// Get raw boolean value
value := MyFeature.Get(ctx)

// Check if the key is set in the context
value, exists := MyFeature.TryGet(ctx)
```

### Working with Value-Based Flags

```go
// Set a value
ctx = MyValueKey.WithValue(ctx, "hello")

// Get the value (returns zero value if not set)
value := MyValueKey.Get(ctx)

// Try to get the value (returns value and bool indicating if set)
value, exists := MyValueKey.TryGet(ctx)
if exists {
    fmt.Printf("Value: %s\n", value)
}
```

## Why Use This Package?

### Problem: Context Key Collisions

When using `context.WithValue` directly with string or int keys, collisions are easy:

```go
// ❌ BAD: These keys will collide!
type key string
var userKey key = "user"
var requestKey key = "user"  // Same underlying value

ctx = context.WithValue(ctx, userKey, "Alice")
ctx = context.WithValue(ctx, requestKey, "Bob")
// userKey now returns "Bob" instead of "Alice"!
```

### Alternative 1: Empty Struct Types (Traditional Approach)

You might think: "Just use `type contextKey struct{}` for each key!"

```go
// 🤔 Avoids collisions, but has other problems
type userIDKey struct{}
type requestIDKey struct{}

var userID = userIDKey{}
var requestID = requestIDKey{}

ctx = context.WithValue(ctx, userID, 123)
ctx = context.WithValue(ctx, requestID, "abc")

// ❌ Type assertions required, can panic
userIDValue := ctx.Value(userID).(int)
requestIDValue := ctx.Value(requestID).(string)
```

**Problems:**
- Need a unique type for each key (boilerplate)
- No compile-time type safety for values
- Requires type assertions everywhere (runtime panics possible)

### Alternative 2: Library Wrapper Around Struct Keys

You might think: "What if a library wrapped those keys and handled assertions?"

```go
// 🤔 Still need to define key types
type userIDKey struct{}
type requestIDKey struct{}

// Library wraps keys with generic type
var UserID = feature.Wrap[int](userIDKey{})
var RequestID = feature.Wrap[string](requestIDKey{})

ctx = UserID.Set(ctx, 123)
value := UserID.Get(ctx)  // Library handles assertion
```

**Problems:**
- Still need to define a unique type for each key (boilerplate)
- **Key type and value type are separate** - nothing prevents:
  ```go
  // In file A
  var UserID = feature.Wrap[int](userIDKey{})

  // In file B (accidentally)
  var UserName = feature.Wrap[string](userIDKey{})  // Same key type, different value type!
  ```
- No compile-time guarantee that key type → value type mapping is consistent
- Two places to define things: the type definition and the var declaration

### This Package's Solution: Type-Safe Pointer Identity + Generics

```go
// ✅ BEST: Collision-free, type-safe, ergonomic
var UserID = feature.New[int]()
var RequestID = feature.New[string]()

ctx = UserID.WithValue(ctx, 123)
ctx = RequestID.WithValue(ctx, "abc")

// ✅ No type assertions, compile-time safety
userIDValue := UserID.Get(ctx)           // int
requestIDValue := RequestID.Get(ctx)     // string

// ✅ Rich API
if UserID.IsSet(ctx) {
    fmt.Println("User ID is set")
}

value, ok := RequestID.TryGet(ctx)
config := SomeKey.GetOrDefault(ctx, defaultValue)
required := RequiredKey.MustGet(ctx)  // Panics with clear message if not set
```

**Benefits:**
1. **Pointer identity**: Each `var` holds a unique pointer, preventing collisions
2. **Type safety**: Generics ensure compile-time type checking
3. **No allocations**: Keys are allocated once as package-level variables
4. **Rich API**: Get, TryGet, GetOrDefault, MustGet, IsSet, IsNotSet, DebugValue
5. **Better debugging**: Named keys show up clearly in logs and error messages
6. **Boolean keys**: Special `BoolKey` type with Enabled/Disabled/ExplicitlyDisabled methods
7. **Three-state logic**: Distinguish between unset, explicitly true, and explicitly false

## Background: Go Official Proposal

A similar idea was proposed to the Go team in 2021:

- [proposal: context: add generic key and value type #49189](https://github.com/golang/go/issues/49189)

The proposal by [@dsnet](https://github.com/dsnet) (Joe Tsai, a Go team member) suggests:

```go
type Key[Value any] struct { name *string }

func NewKey[Value any](name string) Key[Value] {
    return Key[Value]{&name}  // Uses argument address for uniqueness
}

func (k Key[V]) WithValue(ctx Context, val V) Context
func (k Key[V]) Value(ctx Context) (V, bool)
```

This package implements essentially the same concept. However, the official proposal has been on hold for over 3 years, primarily because:

1. **Standard library generics policy is undecided** - [Discussion #48287](https://github.com/golang/go/discussions/48287) is still ongoing about how to add generics to existing packages
2. **Migration path unclear** - Whether to deprecate `context.WithValue`/`context.Value` or keep both APIs
3. **Alternative proposals being considered** - Multiple approaches are being evaluated in parallel

This package provides an immediate, production-ready solution while the Go team deliberates.

## Design Decisions

### Sealed Interface Pattern

Unlike the proposal's struct-based approach, this package uses the **Sealed Interface** pattern:

```go
type Key[V any] interface {
    WithValue(ctx context.Context, value V) context.Context
    Get(ctx context.Context) V
    TryGet(ctx context.Context) (V, bool)
    // ... other methods
    downcast() key[V]  // unexported method prevents external implementation
}

type key[V any] struct {  // unexported implementation
    name  string
    ident *opaque
}
```

**Why this matters:**

The struct-based approach has a subtle vulnerability. In Go, you can bypass constructor functions and directly initialize structs with zero values for unexported fields:

```go
// With struct-based design:
type Key[V any] struct { name *string }

// This compiles! Both keys have nil name pointer
badKeyX := Key[int]{}
badKeyY := Key[string]{}
// These will COLLIDE because both use (*string)(nil) as identity
```

Note: `(*T)(nil)` doesn't panic like `nil` does - it silently uses the zero value as the key, making collisions hard to detect.

With the Sealed Interface pattern:
- The implementation struct `key[V]` is unexported, preventing direct initialization
- The interface contains an unexported method `downcast()`, preventing external implementations
- Users **must** use `feature.New()` or `feature.NewBool()` to create keys

**Additional benefit:** `BoolKey` can be used anywhere `Key[bool]` is expected, providing better interoperability than struct embedding would allow.

### Empty Struct Optimization Avoidance

The internal `opaque` type that provides pointer identity includes a byte field:

```go
type opaque struct {
    _ byte  // Prevents address optimization
}
```

Without this, Go's compiler optimization would give all zero-size struct pointers the same address:

```go
type empty struct{}

a := new(empty)
b := new(empty)
fmt.Printf("%p %p\n", a, b)  // Same address! Keys would collide.
```

## Best Practices

### 1. Define Keys as Package-Level Variables

```go
package myapp

import "github.com/mpyw/feature"

// Define keys at package level to ensure single instance
var (
    EnableBetaFeature = feature.NewNamedBool("beta-feature")
    MaxConcurrency    = feature.New[int]()
)
```

### 2. Use Named Keys for Debugging

```go
// Named keys make debugging easier
var MyFeature = feature.NewNamedBool("my-feature")
fmt.Println(MyFeature)
// Output: my-feature

// Anonymous keys automatically include call site information for debugging
var AnonFeature = feature.NewBool()
fmt.Println(AnonFeature)
// Output: anonymous(/path/to/file.go:42)@0x14000010098
```

### 3. Use Type-Safe Value Keys

```go
// Instead of interface{}, use specific types
var MaxRetries = feature.New[int]()
var UserID = feature.New[string]()
var Config = feature.New[*AppConfig]()
```

## How It Works

Each key holds an internal `*opaque` pointer that serves as its unique identity. This ensures:

1. Each key has a unique identity based on the internal pointer
2. Keys can be used as context keys without collisions
3. Type safety is maintained through generics
4. Even if the key struct is copied, the identity remains the same (copy-safe)

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Related Projects

- [context](https://pkg.go.dev/context) - Go's official context package
- [proposal: context: add generic key and value type #49189](https://github.com/golang/go/issues/49189) - Go official proposal
