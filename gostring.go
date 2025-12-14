package feature

import (
	"fmt"
	"reflect"
)

// GoString returns a Go syntax representation of the key.
// The output is a valid Go expression that creates an equivalent key
// (though with a different identity).
// This implements fmt.GoStringer.
func (k key[V]) GoString() string {
	typeName := reflect.TypeOf((*V)(nil)).Elem().String()

	return fmt.Sprintf("feature.New[%s](feature.WithName(%q))", typeName, k.name)
}

// GoString returns a Go syntax representation of the bool key.
// The output is a valid Go expression that creates an equivalent key
// (though with a different identity).
// This implements fmt.GoStringer.
func (k boolKey) GoString() string {
	return fmt.Sprintf("feature.NewBool(feature.WithName(%q))", k.name)
}
