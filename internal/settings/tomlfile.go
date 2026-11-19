package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

// encodeTables writes agents.toml and prompts.toml: v is a struct whose one
// field is a list of tables ([[agent]], [[prompt]]). A text with line breaks
// is written as a multi-line literal string (between triple quotes), so
// prompts read in the file as agents get them; the TOML library would write
// one line full of \n.
// The defaults in defaults/ are written the same way, so the user's files
// and the defaults compare with a plain diff.
func encodeTables(v any) ([]byte, error) {
	rv := reflect.ValueOf(v)
	f := rv.Type().Field(0)
	name, _ := tomlTag(f)
	list := rv.Field(0)
	var b bytes.Buffer
	for i := 0; i < list.Len(); i++ {
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "[[%s]]\n", name)
		el := list.Index(i)
		for j := 0; j < el.NumField(); j++ {
			key, omitEmpty := tomlTag(el.Type().Field(j))
			fv := el.Field(j)
			if key == "-" || !el.Type().Field(j).IsExported() || (omitEmpty && fv.IsZero()) || (omitEmpty && fv.Kind() == reflect.Slice && fv.Len() == 0) {
				continue
			}
			var val string
			switch fv.Kind() {
			case reflect.String:
				val = tomlString(fv.String())
			case reflect.Bool:
				val = fmt.Sprint(fv.Bool())
			case reflect.Slice:
				items := make([]string, fv.Len())
				for k := range items {
					items[k] = basicString(fv.Index(k).String())
				}
				val = "[" + strings.Join(items, ", ") + "]"
			default:
				return nil, fmt.Errorf("%s.%s: unsupported kind %s", name, key, fv.Kind())
			}
			fmt.Fprintf(&b, "%s = %s\n", key, val)
		}
	}
	return b.Bytes(), nil
}

func tomlTag(f reflect.StructField) (name string, omitEmpty bool) {
	tag := f.Tag.Get("toml")
	name, opts, _ := strings.Cut(tag, ",")
	if name == "" {
		name = f.Name
	}
	return name, opts == "omitempty"
}

// tomlString is a multi-line literal string when the text has line breaks
// and can be one (no triple quote inside, no trailing quote, no control
// characters), else a basic string.
func tomlString(s string) string {
	if !strings.Contains(s, "\n") || strings.Contains(s, "'''") || strings.HasSuffix(s, "'") || strings.ContainsFunc(s, func(r rune) bool { return r < 0x20 && r != '\n' && r != '\t' }) {
		return basicString(s)
	}
	return "'''\n" + s + "'''"
}

// basicString is s as a TOML basic string ("…"): JSON's escapes are valid TOML.
func basicString(s string) string {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	return strings.TrimSuffix(b.String(), "\n")
}
