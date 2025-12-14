package feature_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/mpyw/feature"
)

// TestKey tests the basic functionality of generic Key[V].
func TestKey(t *testing.T) {
	t.Parallel()

	t.Run("unset key returns zero value", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[string]()

		// Get should return zero value (empty string)
		if got := key.Get(ctx); got != "" {
			t.Errorf("Get() = %q, want empty string", got)
		}

		// TryGet should return false
		val, ok := key.TryGet(ctx)
		if ok {
			t.Error("TryGet() ok = true, want false")
		}

		if val != "" {
			t.Errorf("TryGet() value = %q, want empty string", val)
		}
	})

	t.Run("set key returns correct value", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[int]()
		want := 12345

		ctx = key.WithValue(ctx, want)

		// Get should return the set value
		if got := key.Get(ctx); got != want {
			t.Errorf("Get() = %v, want %v", got, want)
		}

		// TryGet should return true and the value
		val, ok := key.TryGet(ctx)
		if !ok {
			t.Error("TryGet() ok = false, want true")
		}

		if val != want {
			t.Errorf("TryGet() value = %v, want %v", val, want)
		}
	})

	t.Run("pointer identity prevents collisions", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		// Create two keys with the same type but different instances
		keyA := feature.New[string]()
		keyB := feature.New[string]()

		// Set different values for each key
		ctx = keyA.WithValue(ctx, "Value-A")
		ctx = keyB.WithValue(ctx, "Value-B")

		// Each key should maintain its own value
		if got := keyA.Get(ctx); got != "Value-A" {
			t.Errorf("keyA.Get() = %q, want %q", got, "Value-A")
		}

		if got := keyB.Get(ctx); got != "Value-B" {
			t.Errorf("keyB.Get() = %q, want %q", got, "Value-B")
		}
	})

	t.Run("context immutability is preserved", func(t *testing.T) {
		t.Parallel()

		ctxBase := context.Background()
		key := feature.New[string]()

		// Create a derived context with a value
		ctxDerived := key.WithValue(ctxBase, "changed")

		// Base context should not be modified
		if _, ok := key.TryGet(ctxBase); ok {
			t.Error("base context was modified, want immutable")
		}

		// Derived context should have the value
		if got := key.Get(ctxDerived); got != "changed" {
			t.Errorf("ctxDerived.Get() = %q, want %q", got, "changed")
		}
	})
}

// TestIsSet tests the IsSet and IsNotSet methods.
func TestIsSet(t *testing.T) {
	t.Parallel()

	t.Run("unset key returns false for IsSet", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[string]()

		if key.IsSet(ctx) {
			t.Error("IsSet() = true, want false for unset key")
		}

		if !key.IsNotSet(ctx) {
			t.Error("IsNotSet() = false, want true for unset key")
		}
	})

	t.Run("set key returns true for IsSet", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[int]()
		ctx = key.WithValue(ctx, 42)

		if !key.IsSet(ctx) {
			t.Error("IsSet() = false, want true for set key")
		}

		if key.IsNotSet(ctx) {
			t.Error("IsNotSet() = true, want false for set key")
		}
	})

	t.Run("zero value is still considered set", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[int]()
		ctx = key.WithValue(ctx, 0) // Explicitly set to zero value

		if !key.IsSet(ctx) {
			t.Error("IsSet() = false, want true even for zero value")
		}

		if key.IsNotSet(ctx) {
			t.Error("IsNotSet() = true, want false even for zero value")
		}
	})
}

// TestGetOrDefault tests the GetOrDefault method.
func TestGetOrDefault(t *testing.T) {
	t.Parallel()

	t.Run("returns default value when key is not set", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[string]()

		defaultValue := "default"
		got := key.GetOrDefault(ctx, defaultValue)

		if got != defaultValue {
			t.Errorf("GetOrDefault() = %q, want %q", got, defaultValue)
		}
	})

	t.Run("returns set value when key is set", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[int]()
		want := 42
		ctx = key.WithValue(ctx, want)

		got := key.GetOrDefault(ctx, 100)

		if got != want {
			t.Errorf("GetOrDefault() = %d, want %d", got, want)
		}
	})

	t.Run("returns zero value over default when explicitly set to zero", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[int]()
		ctx = key.WithValue(ctx, 0) // Explicitly set to zero

		got := key.GetOrDefault(ctx, 100)

		if got != 0 {
			t.Errorf("GetOrDefault() = %d, want 0 (explicitly set zero value)", got)
		}
	})
}

// TestMustGet tests the MustGet method.
func TestMustGet(t *testing.T) {
	t.Parallel()

	t.Run("returns value when key is set", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[string]()
		want := "test-value"
		ctx = key.WithValue(ctx, want)

		got := key.MustGet(ctx)

		if got != want {
			t.Errorf("MustGet() = %q, want %q", got, want)
		}
	})

	t.Run("panics when key is not set", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.NewNamed[string]("required-key")

		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGet() did not panic for unset key")
			} else {
				// Check panic message contains key name
				msg := fmt.Sprint(r)
				if !strings.Contains(msg, "required-key") {
					t.Errorf("panic message %q does not contain key name", msg)
				}

				if !strings.Contains(msg, "is not set in context") {
					t.Errorf("panic message %q does not contain expected text", msg)
				}
			}
		}()

		_ = key.MustGet(ctx) // Should panic
	})

	t.Run("returns zero value when explicitly set to zero", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[int]()
		ctx = key.WithValue(ctx, 0) // Explicitly set to zero

		got := key.MustGet(ctx)

		if got != 0 {
			t.Errorf("MustGet() = %d, want 0", got)
		}
	})
}

// TestBoolKey tests the specialized BoolKey functionality.
func TestBoolKey(t *testing.T) {
	t.Parallel()

	t.Run("unset flag returns false", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewBool()

		if flag.Get(ctx) {
			t.Error("Get() = true, want false for unset flag")
		}

		if flag.Enabled(ctx) {
			t.Error("Enabled() = true, want false for unset flag")
		}

		if !flag.Disabled(ctx) {
			t.Error("Disabled() = false, want true for unset flag")
		}

		if flag.ExplicitlyDisabled(ctx) {
			t.Error("ExplicitlyDisabled() = true, want false for unset flag")
		}
	})

	t.Run("unset flag TryGet returns false", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewBool()

		val, ok := flag.TryGet(ctx)

		if ok {
			t.Error("TryGet() ok = true, want false for unset flag")
		}

		if val {
			t.Error("TryGet() value = true, want false for unset flag")
		}
	})

	t.Run("WithEnabled sets flag to true", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewBool()

		ctx = flag.WithEnabled(ctx)

		if !flag.Get(ctx) {
			t.Error("Get() = false, want true after WithEnabled")
		}

		if !flag.Enabled(ctx) {
			t.Error("Enabled() = false, want true after WithEnabled")
		}

		if flag.Disabled(ctx) {
			t.Error("Disabled() = true, want false after WithEnabled")
		}

		if flag.ExplicitlyDisabled(ctx) {
			t.Error("ExplicitlyDisabled() = true, want false after WithEnabled")
		}

		val, ok := flag.TryGet(ctx)
		if !ok {
			t.Error("TryGet() ok = false, want true after WithEnabled")
		}

		if !val {
			t.Error("TryGet() value = false, want true after WithEnabled")
		}
	})

	t.Run("WithDisabled sets flag to false", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewBool()

		// Explicitly set to disabled
		ctx = flag.WithDisabled(ctx)

		// Value should be false but ok should be true (explicitly set)
		val, ok := flag.TryGet(ctx)

		if !ok {
			t.Error("TryGet() ok = false, want true after WithDisabled")
		}

		if val {
			t.Error("TryGet() value = true, want false after WithDisabled")
		}

		if flag.Enabled(ctx) {
			t.Error("Enabled() = true, want false after WithDisabled")
		}

		if !flag.Disabled(ctx) {
			t.Error("Disabled() = false, want true after WithDisabled")
		}

		if !flag.ExplicitlyDisabled(ctx) {
			t.Error("ExplicitlyDisabled() = false, want true after WithDisabled")
		}
	})

	t.Run("flag can be toggled", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewBool()

		// Enable -> Disable
		ctx = flag.WithEnabled(ctx)
		if !flag.Enabled(ctx) {
			t.Error("Enabled() = false, want true after WithEnabled")
		}

		ctx = flag.WithDisabled(ctx)
		if flag.Enabled(ctx) {
			t.Error("Enabled() = true, want false after WithDisabled")
		}

		// Disable -> Enable
		ctx = flag.WithEnabled(ctx)
		if !flag.Enabled(ctx) {
			t.Error("Enabled() = false, want true after second WithEnabled")
		}
	})

	t.Run("bool keys do not collide", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flagA := feature.NewBool()
		flagB := feature.NewBool()

		ctx = flagA.WithEnabled(ctx)

		// flagB is not touched
		if !flagA.Enabled(ctx) {
			t.Error("flagA.Enabled() = false, want true")
		}

		if flagB.Enabled(ctx) {
			t.Error("flagB.Enabled() = true, want false")
		}
	})

	t.Run("ExplicitlyDisabled distinguishes unset from false", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewBool()

		// Unset state
		if flag.ExplicitlyDisabled(ctx) {
			t.Error("ExplicitlyDisabled() = true for unset flag, want false")
		}

		if !flag.Disabled(ctx) {
			t.Error("Disabled() = false for unset flag, want true")
		}

		// Explicitly disabled
		ctx = flag.WithDisabled(ctx)
		if !flag.ExplicitlyDisabled(ctx) {
			t.Error("ExplicitlyDisabled() = false after WithDisabled, want true")
		}

		if !flag.Disabled(ctx) {
			t.Error("Disabled() = false after WithDisabled, want true")
		}

		// Enabled
		ctx = flag.WithEnabled(ctx)
		if flag.ExplicitlyDisabled(ctx) {
			t.Error("ExplicitlyDisabled() = true after WithEnabled, want false")
		}

		if flag.Disabled(ctx) {
			t.Error("Disabled() = true after WithEnabled, want false")
		}
	})
}

// TestString tests the String method and key name formatting.
func TestString(t *testing.T) {
	t.Parallel()

	// Named key tests

	t.Run("New with WithName returns the name", func(t *testing.T) {
		t.Parallel()

		key := feature.New[string](feature.WithName("test-key"))

		if got := key.String(); got != "test-key" {
			t.Errorf("String() = %q, want %q", got, "test-key")
		}
	})

	t.Run("NewNamed returns the name", func(t *testing.T) {
		t.Parallel()

		key := feature.NewNamed[int]("max-retries")

		if got := key.String(); got != "max-retries" {
			t.Errorf("String() = %q, want %q", got, "max-retries")
		}
	})

	t.Run("NewNamedBool returns the name", func(t *testing.T) {
		t.Parallel()

		flag := feature.NewNamedBool("my-feature")

		if got := flag.String(); got != "my-feature" {
			t.Errorf("String() = %q, want %q", got, "my-feature")
		}
	})
}

// TestNewNamed tests the NewNamed constructor.
func TestNewNamed(t *testing.T) {
	t.Parallel()

	t.Run("creates key with debug name", func(t *testing.T) {
		t.Parallel()

		key := feature.NewNamed[string]("test-key")

		if got := key.String(); got != "test-key" {
			t.Errorf("String() = %q, want %q", got, "test-key")
		}
	})

	t.Run("is equivalent to New with WithName", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		key1 := feature.NewNamed[string]("test")
		key2 := feature.New[string](feature.WithName("test"))

		// Both should have the same string representation
		if key1.String() != key2.String() {
			t.Errorf("NewNamed and New(WithName) produce different strings: %q vs %q",
				key1.String(), key2.String())
		}

		// Both should work independently (different pointer identity)
		ctx = key1.WithValue(ctx, "value1")
		ctx = key2.WithValue(ctx, "value2")

		if got := key1.Get(ctx); got != "value1" {
			t.Errorf("key1.Get() = %q, want %q", got, "value1")
		}

		if got := key2.Get(ctx); got != "value2" {
			t.Errorf("key2.Get() = %q, want %q", got, "value2")
		}
	})
}

// TestComplexTypes tests keys with complex value types.
func TestComplexTypes(t *testing.T) {
	t.Parallel()

	t.Run("struct value type", func(t *testing.T) {
		t.Parallel()

		type Config struct {
			MaxRetries int
			Timeout    string
		}

		key := feature.New[Config]()
		ctx := context.Background()

		want := Config{MaxRetries: 3, Timeout: "30s"}
		ctx = key.WithValue(ctx, want)

		if got := key.Get(ctx); got != want {
			t.Errorf("Get() = %+v, want %+v", got, want)
		}
	})

	t.Run("pointer value type", func(t *testing.T) {
		t.Parallel()

		type User struct {
			ID   int
			Name string
		}

		key := feature.New[*User]()
		ctx := context.Background()

		want := &User{ID: 123, Name: "Alice"}
		ctx = key.WithValue(ctx, want)

		got := key.Get(ctx)
		if got != want {
			t.Errorf("Get() = %p, want %p", got, want)
		}

		if got.ID != want.ID || got.Name != want.Name {
			t.Errorf("Get() = %+v, want %+v", got, want)
		}
	})

	t.Run("slice value type", func(t *testing.T) {
		t.Parallel()

		key := feature.New[[]string]()
		ctx := context.Background()
		want := []string{"a", "b", "c"}
		ctx = key.WithValue(ctx, want)
		got := key.Get(ctx)

		if len(got) != len(want) {
			t.Fatalf("Get() length = %d, want %d", len(got), len(want))
		}

		for idx := range want {
			if got[idx] != want[idx] {
				t.Errorf("Get()[%d] = %q, want %q", idx, got[idx], want[idx])
			}
		}
	})
}

// TestConcurrency tests that keys are safe for concurrent use.
func TestConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("concurrent reads and writes", func(t *testing.T) {
		t.Parallel()

		key := feature.New[int]()
		ctx := context.Background()

		// Simulate concurrent goroutines using the same key
		done := make(chan bool)

		for idx := 0; idx < 10; idx++ {
			go func(value int) {
				localCtx := key.WithValue(ctx, value)
				got := key.Get(localCtx)

				if got != value {
					t.Errorf("concurrent Get() = %d, want %d", got, value)
				}

				done <- true
			}(idx)
		}

		for idx := 0; idx < 10; idx++ {
			<-done
		}
	})
}

// =============================================================================
// Example Tests - Organized by Implementation Order
// =============================================================================

// Constructor Examples

func ExampleNewBool() {
	ctx := context.Background()

	// Create a boolean feature flag
	var EnableNewUI = feature.NewBool()

	// Enable the feature
	ctx = EnableNewUI.WithEnabled(ctx)

	// Check if enabled
	if EnableNewUI.Enabled(ctx) {
		fmt.Println("New UI is enabled")
	}

	// Output:
	// New UI is enabled
}

func ExampleNewNamedBool() {
	// Create a named boolean feature flag for easier debugging
	var EnableBetaFeature = feature.NewNamedBool("beta-feature")

	fmt.Println(EnableBetaFeature)

	// Output:
	// beta-feature
}

func ExampleNew() {
	ctx := context.Background()

	// Create a feature flag with a custom type
	var MaxRetries = feature.New[int]()

	// Set a value
	ctx = MaxRetries.WithValue(ctx, 5)

	// Retrieve the value
	retries := MaxRetries.Get(ctx)
	fmt.Printf("Max retries: %d\n", retries)

	// Output:
	// Max retries: 5
}

func ExampleNewNamed() {
	ctx := context.Background()

	// Create a named feature flag with a custom type
	var MaxRetries = feature.NewNamed[int]("max-retries")

	// Set a value
	ctx = MaxRetries.WithValue(ctx, 3)

	// Retrieve the value and name
	fmt.Printf("%s: %d\n", MaxRetries, MaxRetries.Get(ctx))

	// Output:
	// max-retries: 3
}

// Option Examples

func ExampleWithName() {
	// Use WithName option to create a key with a debug name
	var APIKey = feature.New[string](
		feature.WithName("api-key"),
	)

	fmt.Println(APIKey)

	// Output:
	// api-key
}

// Key[V] Method Examples

func ExampleKey_WithValue() {
	ctx := context.Background()

	var Timeout = feature.New[int]()

	// Add value to context
	ctx = Timeout.WithValue(ctx, 30)

	fmt.Printf("Timeout: %d seconds\n", Timeout.Get(ctx))

	// Output:
	// Timeout: 30 seconds
}

func ExampleKey_Get() {
	ctx := context.Background()

	var Port = feature.New[int]()

	// Get returns zero value if not set
	fmt.Printf("Default port: %d\n", Port.Get(ctx))

	// Set a value
	ctx = Port.WithValue(ctx, 8080)
	fmt.Printf("Configured port: %d\n", Port.Get(ctx))

	// Output:
	// Default port: 0
	// Configured port: 8080
}

func ExampleKey_TryGet() {
	ctx := context.Background()

	var MaxItems = feature.New[int]()

	// Check if the key is set
	if value, ok := MaxItems.TryGet(ctx); ok {
		fmt.Printf("Max items: %d\n", value)
	} else {
		fmt.Println("Max items not set")
	}

	// Set a value and check again
	ctx = MaxItems.WithValue(ctx, 100)
	if value, ok := MaxItems.TryGet(ctx); ok {
		fmt.Printf("Max items: %d\n", value)
	}

	// Output:
	// Max items not set
	// Max items: 100
}

func ExampleKey_GetOrDefault() {
	ctx := context.Background()

	var MaxItems = feature.New[int]()

	// Get with default when not set
	limit := MaxItems.GetOrDefault(ctx, 50)
	fmt.Printf("Limit: %d\n", limit)

	// Set a value
	ctx = MaxItems.WithValue(ctx, 100)
	limit = MaxItems.GetOrDefault(ctx, 50)
	fmt.Printf("Limit: %d\n", limit)

	// Output:
	// Limit: 50
	// Limit: 100
}

func ExampleKey_MustGet() {
	ctx := context.Background()

	var RequiredConfig = feature.New[string]()

	// Set a required value
	ctx = RequiredConfig.WithValue(ctx, "production")

	// Get the value (will not panic because it's set)
	config := RequiredConfig.MustGet(ctx)
	fmt.Printf("Config: %s\n", config)

	// Output:
	// Config: production
}

func ExampleKey_IsSet() {
	ctx := context.Background()

	var Timeout = feature.New[int]()

	// Check if timeout is set
	if Timeout.IsSet(ctx) {
		fmt.Printf("Timeout is set to: %d\n", Timeout.Get(ctx))
	} else {
		fmt.Println("Timeout is not set")
	}

	// Set timeout to 30
	ctx = Timeout.WithValue(ctx, 30)
	if Timeout.IsSet(ctx) {
		fmt.Printf("Timeout is set to: %d\n", Timeout.Get(ctx))
	}

	// Even zero value is considered "set"
	ctx = Timeout.WithValue(ctx, 0)
	if Timeout.IsSet(ctx) {
		fmt.Printf("Timeout is explicitly set to: %d\n", Timeout.Get(ctx))
	}

	// Output:
	// Timeout is not set
	// Timeout is set to: 30
	// Timeout is explicitly set to: 0
}

func ExampleKey_IsNotSet() {
	ctx := context.Background()

	var CacheSize = feature.New[int]()

	// Check if not set
	if CacheSize.IsNotSet(ctx) {
		fmt.Println("Using default cache size")
	}

	// Set a value
	ctx = CacheSize.WithValue(ctx, 1024)
	if CacheSize.IsNotSet(ctx) {
		fmt.Println("Using default cache size")
	} else {
		fmt.Printf("Using cache size: %d\n", CacheSize.Get(ctx))
	}

	// Output:
	// Using default cache size
	// Using cache size: 1024
}

func ExampleKey_Inspect() {
	ctx := context.Background()

	// Create a named key for better debug output
	var MaxRetries = feature.NewNamed[int]("max-retries")

	// Inspect when not set
	fmt.Println(MaxRetries.Inspect(ctx))

	// Set a value and inspect again
	ctx = MaxRetries.WithValue(ctx, 5)
	inspection := MaxRetries.Inspect(ctx)
	fmt.Println(inspection)
	fmt.Println("Value:", inspection.Get())
	fmt.Println("Is set:", inspection.IsSet())

	// Output:
	// max-retries: <not set>
	// max-retries: 5
	// Value: 5
	// Is set: true
}

func ExampleKey_String() {
	// WithName keys show their name
	namedKey := feature.NewNamed[string]("api-key")
	fmt.Println(namedKey)

	// Anonymous keys show their call site info and address
	anonymousKey := feature.New[int]()
	str := anonymousKey.String()
	// Check that it starts with "anonymous(" (call site info is included)
	if strings.HasPrefix(str, "anonymous(") && strings.Contains(str, "@0x") {
		fmt.Println("Anonymous key format: anonymous(file:line)@<address>")
	}

	// Output:
	// api-key
	// Anonymous key format: anonymous(file:line)@<address>
}

// BoolKey Method Examples

func ExampleBoolKey_Enabled() {
	ctx := context.Background()

	var DebugMode = feature.NewNamedBool("debug-mode")

	// Check if debug mode is enabled (default: false)
	if DebugMode.Enabled(ctx) {
		fmt.Println("Debug mode is on")
	} else {
		fmt.Println("Debug mode is off")
	}

	// Enable debug mode
	ctx = DebugMode.WithEnabled(ctx)
	if DebugMode.Enabled(ctx) {
		fmt.Println("Debug mode is on")
	}

	// Output:
	// Debug mode is off
	// Debug mode is on
}

func ExampleBoolKey_Disabled() {
	ctx := context.Background()

	var MaintenanceMode = feature.NewNamedBool("maintenance")

	// Disabled returns true if the feature is either not set or explicitly disabled
	if MaintenanceMode.Disabled(ctx) {
		fmt.Println("System is operational")
	}

	// Explicitly disable
	ctx = MaintenanceMode.WithDisabled(ctx)
	if MaintenanceMode.Disabled(ctx) {
		fmt.Println("System is in maintenance")
	}

	// Output:
	// System is operational
	// System is in maintenance
}

func ExampleBoolKey_ExplicitlyDisabled() {
	ctx := context.Background()

	var BetaFeature = feature.NewNamedBool("beta")

	// Three states: unset, enabled, and explicitly disabled.
	switch {
	case BetaFeature.Enabled(ctx):
		fmt.Println("Beta feature is enabled")
	case BetaFeature.ExplicitlyDisabled(ctx):
		fmt.Println("Beta feature is explicitly disabled")
	default:
		fmt.Println("Beta feature is not set (use default behavior)")
	}

	// Explicitly disable
	ctx = BetaFeature.WithDisabled(ctx)

	switch {
	case BetaFeature.Enabled(ctx):
		fmt.Println("Beta feature is enabled")
	case BetaFeature.ExplicitlyDisabled(ctx):
		fmt.Println("Beta feature is explicitly disabled")
	default:
		fmt.Println("Beta feature is not set (use default behavior)")
	}

	// Output:
	// Beta feature is not set (use default behavior)
	// Beta feature is explicitly disabled
}

func ExampleBoolKey_WithEnabled() {
	ctx := context.Background()

	var FeatureFlag = feature.NewNamedBool("new-feature")

	// Enable the feature
	ctx = FeatureFlag.WithEnabled(ctx)

	if FeatureFlag.Enabled(ctx) {
		fmt.Println("Feature is enabled")
	}

	// Output:
	// Feature is enabled
}

func ExampleBoolKey_WithDisabled() {
	ctx := context.Background()

	var FeatureFlag = feature.NewNamedBool("experimental")

	// Explicitly disable the feature
	ctx = FeatureFlag.WithDisabled(ctx)

	if FeatureFlag.ExplicitlyDisabled(ctx) {
		fmt.Println("Feature is explicitly disabled")
	}

	// Output:
	// Feature is explicitly disabled
}
