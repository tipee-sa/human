package human

import (
	"strings"
	"testing"
	"time"
)

func MustRender(t *testing.T, value any) string {
	str, err := Render(value)
	if err != nil {
		t.Fatalf("rendering failed: %s", err)
	}
	return str
}

func AssertRender(t *testing.T, value any, expected string) {
	str := MustRender(t, value)
	if str == "" && expected == "" {
		return
	}
	if str != expected+"\n" {
		t.Fatalf("expected %q, got %q", expected, str)
	}
}

func lines(lines ...string) string {
	return strings.Join(lines, "\n")
}

func TestNil(t *testing.T) {
	AssertRender(t, nil, "")
}

func TestBool(t *testing.T) {
	AssertRender(t, false, "false")
	AssertRender(t, true, "true")
}

func TestString(t *testing.T) {
	AssertRender(t, "hello", "hello")
}

func TestNumbers(t *testing.T) {
	AssertRender(t, 42, "42")
	AssertRender(t, 3.14, "3.14")
	AssertRender(t, -100, "-100")
}

func TestTime(t *testing.T) {
	now := time.Now()
	AssertRender(t, now, now.String())
}

func TestDuration(t *testing.T) {
	AssertRender(t, time.Second, "1s")
	AssertRender(t, time.Minute, "1m0s")
	AssertRender(t, time.Hour, "1h0m0s")
}

func TestStruct(t *testing.T) {
	// Simple struct
	type A struct {
		A  int
		Bb string
	}
	AssertRender(t, A{42, "hello"}, lines(
		" A : 42   ",
		"Bb : hello",
	))

	// Nil field
	type B struct {
		A   int
		Nil *int
	}
	AssertRender(t, B{42, nil}, lines(
		"A : 42",
	))

	// Skip field
	type C struct {
		A int
		B bool `human:"skip-field"`
	}
	AssertRender(t, C{42, true}, lines(
		"A : 42",
	))

	// Unexported field
	type D struct {
		A int
		b bool
	}
	AssertRender(t, D{42, true}, lines(
		"A : 42",
	))

	type E struct {
		B    int
		Cdef string
	}
	type F struct {
		A E
	}
	AssertRender(t, F{E{42, "hello"}}, lines(
		"A :    B : 42   ",
		"    Cdef : hello",
		"                ",
	))
}

func TestSlice(t *testing.T) {
	AssertRender(t, []int{1, 2, 3}, lines("1", "2", "3"))
	AssertRender(t, []bool{true, false}, lines("true", "false"))

	now := time.Now()
	nowStr := now.String()
	AssertRender(t, []time.Time{now, now}, lines(nowStr, nowStr))

	type A struct {
		Alligator int       `human:"skip-column"` // Skip column
		Beaver    string    // Simple
		Camel     bool      `human:"skip-field"` // Ignored in a column context
		duck      time.Time // Unexported
	}
	AssertRender(t, []A{{1, "hello", true, now}, {2, "world", false, now}}, lines(
		"BEAVER  CAMEL    ",
		"hello   true    ",
		"world   false   ",
	))
}
