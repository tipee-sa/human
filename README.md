# Human

Renders just about anything as human-friendly as possible.

Primary use case is as a output formatter for CLIs tools.

## Usage

```go
var value any
str, err := human.Render(value)
```

```go
var value any
err := human.Write(os.Stdout, value)
```

```go
var value any
err := human.NewRenderer(os.Stdout).Render(value)
```

## Custom rendering

Render functions should write a new-line at the end of the output.

### Using the `human.RenderHuman` interface

```go
type MyType struct {
    // ...
}

func (t *MyType) RenderHuman(io.Writer) error {
    // ...
}
```

### Registering a type renderer

```go
renderer := human.NewRenderer(os.Stdout)

RegisterTypeRenderer(renderer, func (io.Writer, b bool) error {
    // ...
})

var b bool
err := renderer.Render(b)
```
