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

func replace(re *regexp.Regexp, text string, repl func(match []string) string) string {
	return re.ReplaceAllStringFunc(text, func(s string) string {
		match := re.FindStringSubmatch(s)
		return repl(match)
	})
}

func Process(text string, values Values) (string, error) {
	text = replace(reIfdef, text, func(s []string) string {
		cond := s[1]
		return fmt.Sprintf("{{- if ifdef .Values %q }}", cond)
	})

	text = replace(reIfndef, text, func(s []string) string {
		cond := s[1]
		return fmt.Sprintf("{{- if ifndef .Values %q }}", cond)
	})

	text = replace(reElseIfdef, text, func(s []string) string {
		cond := s[1]
		return fmt.Sprintf("{{- else if ifdef .Values %q }}", cond)
	})

	text = replace(reElse, text, func(_ []string) string {
		return "{{- else }}"
	})

	text = replace(reEnd, text, func(_ []string) string {
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
var reElseIfdef = regexp.MustCompile(`(?m)^\s*#else\s+ifdef\s+(.*)$`)
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
