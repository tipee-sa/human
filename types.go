package human

import (
	"io"
	"reflect"
)

func RegisterTypeRenderer[T any](renderer Renderer, fn RenderFn[T]) {
	t := reflect.TypeFor[T]()
	renderer.typeRenders[t] = func(w io.Writer, v reflect.Value) error {
		return fn(w, v.Interface().(T))
	}
}
