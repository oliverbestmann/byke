package pre

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/oliverbestmann/byke/internal/set"
)

var ErrNoUniqueImportPath = errors.New("#module must be defined exactly once")

type Values map[string]string

func (v Values) Define(name string, defined bool) {
	if defined {
		v[name] = ""
	} else {
		delete(v, name)
	}
}

func (v Values) Set(name string, value string) {
	v[name] = value
}

type Compiler struct {
	files *template.Template
}

func New() Compiler {
	return Compiler{files: template.New("_lib").Funcs(funcs)}
}

func (c *Compiler) MustAdd(source string) {
	if err := c.Add(source); err != nil {
		panic(err)
	}
}

func (c *Compiler) Add(source string) error {
	matches := reModule.FindAllStringSubmatch(source, 2)
	if len(matches) != 1 {
		return ErrNoUniqueImportPath
	}

	// the module name
	mod := matches[0][1]

	// need to translate it to templates
	source = prepareSource(source)

	_, err := c.files.New(mod).Parse(source)
	if err != nil {
		return fmt.Errorf("parse module %q: %w", mod, err)
	}

	slog.Info("Added shader library", slog.String("module", mod))

	return nil
}

func (c *Compiler) PreCompile(source string, values Values) (string, error) {
	if !strings.Contains(source, "#") {
		return source, nil
	}

	source = prepareSource(source)

	t, err := template.Must(c.files.Clone()).New("_source").Parse(source)
	if err != nil {
		return "", fmt.Errorf("pre-parse shader: %w", err)
	}

	// expose the values as .Values
	v := map[string]any{
		"Values":  values,
		"Imports": &set.Set[string]{},
	}

	// compile the template
	var w strings.Builder
	if err := t.Execute(&w, v); err != nil {
		return "", fmt.Errorf("pre-compile shader: %w", err)
	}

	result := w.String()

	// replace the defined values by their value
	keysByLengthDesc := slices.SortedFunc(
		maps.Keys(values),
		func(a, b string) int { return len(b) - len(a) },
	)

	for _, key := range keysByLengthDesc {
		re := regexp.MustCompile("\\b" + regexp.QuoteMeta(key) + "\\b")
		result = re.ReplaceAllLiteralString(result, values[key])
	}

	return result, nil
}
