package human

import (
	"bytes"
	"io"
	"reflect"
)

func Render(value any) (string, error) {
	var buf bytes.Buffer
	if err := NewRenderer(&buf).Render(value); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func Write(w io.Writer, value any) error {
	return NewRenderer(w).Render(value)
}

type RenderHuman interface {
	RenderHuman(io.Writer) error
}

type RenderFn[T any] func(io.Writer, T) error

type typeRendersMap map[reflect.Type]func(io.Writer, reflect.Value) error
