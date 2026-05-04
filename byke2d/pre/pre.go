package pre

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

type Values map[string]string

func (v Values) Define(name string, defined bool) {
	if defined {
		v[name] = ""
	} else {
		delete(v, name)
	}
}

func Process(text string, values Values) (string, error) {
	text = reIfdef.ReplaceAllStringFunc(text, func(s string) string {
		text = strings.TrimSpace(text)
		cond := strings.TrimPrefix(text, "#ifdef ")
		return fmt.Sprintf("{{- if ifdef .Values %q }}", cond)
	})

	text = reIfndef.ReplaceAllStringFunc(text, func(s string) string {
		text = strings.TrimSpace(text)
		cond := strings.TrimPrefix(text, "#ifndef ")
		return fmt.Sprintf("{{- if ifndef .Values %q }}", cond)
	})

	text = reElse.ReplaceAllStringFunc(text, func(s string) string {
		return "{{- else }}"
	})

	text = reEnd.ReplaceAllStringFunc(text, func(s string) string {
		return "{{- end }}"
	})

	tmpl := template.New("").Funcs(funcs)

	if _, err := tmpl.Parse(text); err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}

	var out strings.Builder
	err := tmpl.Execute(&out, map[string]any{"Values": values})
	if err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return out.String(), nil
}

var reIfdef = regexp.MustCompile(`(?m)^\s*#ifdef\s+(.*)$`)
var reIfndef = regexp.MustCompile(`(?m)^\s*#ifndef\s+(.*)$`)
var reElse = regexp.MustCompile(`(?m)^\s*#else\s*$`)
var reEnd = regexp.MustCompile(`(?m)^\s*#endif\s*$`)

var funcs = template.FuncMap{
	"ifdef": func(values Values, cond string) bool {
		_, ok := values[cond]
		return ok
	},
	"ifndef": func(values Values, cond string) bool {
		_, ok := values[cond]
		return !ok
	},
}
