package human

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
	"unicode"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

const (
	humanTag = "human"
)

var (
	stringerType = reflect.TypeFor[fmt.Stringer]()
)

type renderContext int

const (
	contextField renderContext = iota
	contextColumn
)

type Renderer struct {
	out         io.Writer
	typeRenders typeRendersMap
}

func NewRenderer(out io.Writer) *Renderer {
	return &Renderer{out: out, typeRenders: make(typeRendersMap)}
}

func (r *Renderer) Render(value any) error {
	return r.renderHuman(value, false)
}

func (r *Renderer) renderHuman(value any, skipColumn bool) error {
	if display, ok := value.(RenderHuman); ok {
		return display.RenderHuman(r.out)
	}

	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return nil
	}
	for {
		t := v.Type()

		// Check if we have a custom renderer for this type
		if typeRenderer, ok := r.typeRenders[t]; ok {
			return typeRenderer(r.out, v)
		}

		if t.Implements(stringerType) {
			return r.renderHuman(v.Interface().(fmt.Stringer).String(), false)
		}

		// Render based on the type kind
		switch t.Kind() {
		case reflect.Ptr:
			if v.IsZero() {
				return nil
			}
			v = v.Elem()
			continue

		case reflect.Slice:
			elem := t.Elem()
			for elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			if typeRenderer, ok := r.typeRenders[elem]; ok {
				for i := 0; i < v.Len(); i++ {
					if err := typeRenderer(r.out, v.Index(i)); err != nil {
						return err
					}
				}
			} else if elem.Kind() == reflect.Struct && !elem.Implements(stringerType) {
				return r.renderStructList(v, elem, skipColumn)
			} else {
				for i := 0; i < v.Len(); i++ {
					if err := r.renderHuman(v.Index(i).Interface(), false); err != nil {
						return err
					}
				}
			}
			return nil

		case reflect.Struct:
			return r.renderStruct(v, t)

		case reflect.String:
			_, err := fmt.Fprintln(r.out, v.String())
			return err
		}

		// No more hope, let's render as YAML
		return r.renderAsYaml(value)
	}
}

func (r *Renderer) renderStructList(value reflect.Value, tpe reflect.Type, skipHeader bool) error {
	table := tablewriter.NewWriter(r.out)
	table.SetNoWhiteSpace(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetAutoWrapText(false)

	headers := make([]string, 0, tpe.NumField())
	cols := tpe.NumField()
	for i := 0; i < cols; i++ {
		field := tpe.Field(i)
		if shouldDisplayField(field, contextColumn) {
			headers = append(headers, field.Name+"  ")
		}
	}
	if !skipHeader {
		table.SetHeader(headers)
	}

	rows := value.Len()
	for i := 0; i < rows; i++ {
		row := make([]string, len(headers))
		item := value.Index(i)
		for item.Type().Kind() == reflect.Ptr {
			item = item.Elem()
		}
		for i, col := 0, 0; i < cols; i++ {
			field := tpe.Field(i)
			if shouldDisplayField(field, contextColumn) {
				var value string
				var err error

				if field.Type.Kind() == reflect.Slice && strings.Contains(field.Tag.Get(humanTag), "inline") {
					for j, end := 0, item.Field(i).Len(); j < end; j++ {
						if j > 0 {
							value += ", "
						}
						var str string
						str, err = renderHumanValue(item.Field(i).Index(j).Interface())
						if err != nil {
							break
						}
						value += str
					}
				} else {
					value, err = renderHumanValue(item.Field(i).Interface())
				}

				if err != nil {
					return err
				}

				row[col] = value + "   "
				col += 1
			}
		}
		table.Append(row)
	}

	table.Render()
	return nil
}

func (r *Renderer) renderStruct(v reflect.Value, t reflect.Type) error {
	table := tablewriter.NewWriter(r.out)
	table.SetNoWhiteSpace(true)
	table.SetBorder(false)
	table.SetColumnAlignment([]int{tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_LEFT})
	table.SetAutoWrapText(false)

	fields := t.NumField()
	hasField := false
	for i := 0; i < fields; i++ {
		field := t.Field(i)
		value := v.Field(i)
		if shouldDisplayField(field, contextField) && shouldDisplayValue(value) {
			value, err := renderHumanValue(value.Interface())
			if err != nil {
				return err
			}

			hasField = true
			table.Append([]string{field.Name + " : ", value})
		}
	}

	if hasField {
		table.Render()
		return nil
	} else {
		_, err := fmt.Fprintln(r.out, v.Interface())
		return err
	}
}

func (r *Renderer) renderAsYaml(value any) error {
	// First encoding/decoding as JSON to strip all type information that might
	// affect the YAML output. We only want "yaml-y" style formatting.
	json, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error marshaling into JSON: %w", err)
	}

	var jsonObj any
	if err = yaml.Unmarshal(json, &jsonObj); err != nil {
		return fmt.Errorf("error converting JSON to YAML: %w", err)
	}

	enc := yaml.NewEncoder(r.out)
	enc.SetIndent(2)
	return enc.Encode(jsonObj)
}

func renderHumanValue(value any) (string, error) {
	str, err := Render(value)
	if err != nil {
		return "", err
	}
	str = strings.TrimRightFunc(str, unicode.IsSpace)
	if strings.Contains(str, "\n") {
		str = str + "\n"
	}
	return str, nil
}

func shouldDisplayField(field reflect.StructField, context renderContext) bool {
	// Check human tag
	tag := field.Tag.Get(humanTag)
	if context == contextColumn {
		if strings.Contains(tag, "skip-column") {
			return false
		}
	} else if context == contextField {
		if strings.Contains(tag, "skip-field") {
			return false
		}
	}

	// Fallback to Go visibility rule
	return field.IsExported()
}

func shouldDisplayValue(v reflect.Value) bool {
	switch v.Type().Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return !v.IsNil()
	default:
		return true
	}
}
