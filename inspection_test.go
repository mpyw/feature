package feature_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mpyw/feature"
)

// checkContains is a helper to check if got contains want.
func checkContains(t *testing.T, got, want string) {
	t.Helper()

	if !strings.Contains(got, want) {
		t.Errorf("got %q, want to contain %q", got, want)
	}
}

func TestInspectionString(t *testing.T) {
	t.Parallel()

	t.Run("unset key shows not set", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.NewNamed[string]("test-key")

		inspection := key.Inspect(ctx)

		if inspection.Ok {
			t.Error("Inspection.Ok = true, want false")
		}

		want := "test-key: <not set>"
		if got := inspection.String(); got != want {
			t.Errorf("Inspection.String() = %q, want %q", got, want)
		}
	})

	t.Run("set key shows name and value", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.NewNamed[int]("max-retries")
		ctx = key.WithValue(ctx, 5)
		inspection := key.Inspect(ctx)

		if !inspection.Ok {
			t.Error("Inspection.Ok = false, want true")
		}

		if inspection.Value != 5 {
			t.Errorf("Inspection.Value = %d, want 5", inspection.Value)
		}

		want := "max-retries: 5"
		if got := inspection.String(); got != want {
			t.Errorf("Inspection.String() = %q, want %q", got, want)
		}
	})

	t.Run("anonymous key shows call site info in name", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.New[string]()
		ctx = key.WithValue(ctx, "value")

		inspection := key.Inspect(ctx)
		str := inspection.String()
		// Should contain "anonymous(" (call site info) and "@0x" (address) and ": value"
		checkContains(t, str, "anonymous(")
		checkContains(t, str, "@0x")
		checkContains(t, str, ": value")
	})

	t.Run("complex value types are formatted", func(t *testing.T) {
		t.Parallel()

		type Config struct {
			MaxRetries int
			Timeout    string
		}

		ctx := context.Background()
		key := feature.NewNamed[Config]("config")
		ctx = key.WithValue(ctx, Config{MaxRetries: 3, Timeout: "30s"})

		inspection := key.Inspect(ctx)
		str := inspection.String()
		// Should contain the key name and struct representation
		checkContains(t, str, "config:")
		// Check that it contains the struct values
		checkContains(t, str, "3")
		checkContains(t, str, "30s")
	})
}

func TestBoolInspectionString(t *testing.T) {
	t.Parallel()

	t.Run("unset", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewNamedBool("enable-feature")
		inspection := flag.InspectBool(ctx)
		want := "enable-feature: <not set>"

		if got := inspection.String(); got != want {
			t.Errorf("BoolInspection.String() unset = %q, want %q", got, want)
		}
	})

	t.Run("enabled", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewNamedBool("enable-feature")
		ctx = flag.WithEnabled(ctx)
		inspection := flag.InspectBool(ctx)
		want := "enable-feature: true"

		if got := inspection.String(); got != want {
			t.Errorf("BoolInspection.String() enabled = %q, want %q", got, want)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewNamedBool("enable-feature")
		ctx = flag.WithDisabled(ctx)
		inspection := flag.InspectBool(ctx)
		want := "enable-feature: false"

		if got := inspection.String(); got != want {
			t.Errorf("BoolInspection.String() disabled = %q, want %q", got, want)
		}
	})
}

func TestInspectionHelperMethods(t *testing.T) {
	t.Parallel()

	t.Run("set key", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.NewNamed[int]("test")
		ctx = key.WithValue(ctx, 42)

		inspection := key.Inspect(ctx)

		// Test Get
		if got := inspection.Get(); got != 42 {
			t.Errorf("Inspection.Get() = %d, want 42", got)
		}

		// Test TryGet
		val, ok := inspection.TryGet()
		if !ok || val != 42 {
			t.Errorf("Inspection.TryGet() = (%d, %v), want (42, true)", val, ok)
		}

		// Test GetOrDefault
		if got := inspection.GetOrDefault(100); got != 42 {
			t.Errorf("Inspection.GetOrDefault(100) = %d, want 42", got)
		}

		// Test IsSet
		if !inspection.IsSet() {
			t.Error("Inspection.IsSet() = false, want true")
		}

		// Test IsNotSet
		if inspection.IsNotSet() {
			t.Error("Inspection.IsNotSet() = true, want false")
		}
	})

	t.Run("unset key", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.NewNamed[int]("test")

		inspection := key.Inspect(ctx)

		// Test Get returns zero value
		if got := inspection.Get(); got != 0 {
			t.Errorf("Inspection.Get() = %d, want 0", got)
		}

		// Test TryGet returns not ok
		val, ok := inspection.TryGet()
		if ok || val != 0 {
			t.Errorf("Inspection.TryGet() = (%d, %v), want (0, false)", val, ok)
		}

		// Test GetOrDefault returns default
		if got := inspection.GetOrDefault(100); got != 100 {
			t.Errorf("Inspection.GetOrDefault(100) = %d, want 100", got)
		}

		// Test IsSet
		if inspection.IsSet() {
			t.Error("Inspection.IsSet() = true, want false")
		}

		// Test IsNotSet
		if !inspection.IsNotSet() {
			t.Error("Inspection.IsNotSet() = false, want true")
		}
	})

	t.Run("MustGet panics for unset key", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.NewNamed[int]("panic-test")

		inspection := key.Inspect(ctx)

		defer func() {
			if r := recover(); r == nil {
				t.Error("MustGet() did not panic for unset key")
			}
		}()

		inspection.MustGet()
	})
}

func TestBoolInspectionHelperMethods(t *testing.T) {
	t.Parallel()

	t.Run("unset flag", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewNamedBool("test")

		inspection := flag.InspectBool(ctx)
		if inspection.Enabled() {
			t.Error("BoolInspection.Enabled() = true for unset flag, want false")
		}

		if !inspection.Disabled() {
			t.Error("BoolInspection.Disabled() = false for unset flag, want true")
		}

		if inspection.ExplicitlyDisabled() {
			t.Error("BoolInspection.ExplicitlyDisabled() = true for unset flag, want false")
		}
	})

	t.Run("enabled flag", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewNamedBool("test")
		ctx = flag.WithEnabled(ctx)

		inspection := flag.InspectBool(ctx)
		if !inspection.Enabled() {
			t.Error("BoolInspection.Enabled() = false for enabled flag, want true")
		}

		if inspection.Disabled() {
			t.Error("BoolInspection.Disabled() = true for enabled flag, want false")
		}
	})

	t.Run("explicitly disabled flag", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewNamedBool("test")
		ctx = flag.WithDisabled(ctx)

		inspection := flag.InspectBool(ctx)
		if inspection.Enabled() {
			t.Error("BoolInspection.Enabled() = true for disabled flag, want false")
		}

		if !inspection.ExplicitlyDisabled() {
			t.Error("BoolInspection.ExplicitlyDisabled() = false for disabled flag, want true")
		}
	})
}

func TestInspectionGoString(t *testing.T) {
	t.Parallel()

	t.Run("Inspection format", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		key := feature.NewNamed[string]("test-key")

		// Unset
		inspection := key.Inspect(ctx)

		goStr := inspection.GoString()
		checkContains(t, goStr, "feature.Inspection[string]")
		checkContains(t, goStr, "test-key")
		checkContains(t, goStr, "Ok: false")

		// Set
		ctx = key.WithValue(ctx, "hello")
		inspection = key.Inspect(ctx)

		goStr = inspection.GoString()
		checkContains(t, goStr, "feature.Inspection[string]")
		checkContains(t, goStr, "test-key")
		checkContains(t, goStr, "hello")
		checkContains(t, goStr, "Ok: true")
	})

	t.Run("BoolInspection format", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		flag := feature.NewNamedBool("test-flag")

		// Unset
		inspection := flag.InspectBool(ctx)

		goStr := inspection.GoString()
		checkContains(t, goStr, "feature.BoolInspection")
		checkContains(t, goStr, "test-flag")
		checkContains(t, goStr, "Ok: false")

		// Set to true
		ctx = flag.WithEnabled(ctx)
		inspection = flag.InspectBool(ctx)

		goStr = inspection.GoString()
		checkContains(t, goStr, "feature.BoolInspection")
		checkContains(t, goStr, "test-flag")
		checkContains(t, goStr, "Value: true")
		checkContains(t, goStr, "Ok: true")
	})
}
