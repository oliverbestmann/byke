package pre

import (
	"fmt"
	"regexp"
	"text/template"

	"github.com/oliverbestmann/byke/internal/set"
)

var reIfdef = regexp.MustCompile(`(?m)^\s*#ifdef\s+(.*)$`)
var reElseIfdef = regexp.MustCompile(`(?m)^\s*#else\s+ifdef\s+(.*)$`)
var reIfndef = regexp.MustCompile(`(?m)^\s*#ifndef\s+(.*)$`)
var reElse = regexp.MustCompile(`(?m)^\s*#else\s*$`)
var reEnd = regexp.MustCompile(`(?m)^\s*#endif\s*$`)

var reModule = regexp.MustCompile(`(?m)^#module\s+([a-zA-Z][a-zA-Z0-9_]*(?:::[a-zA-Z][a-zA-Z0-9_]*)*)\s*$`)
var reImport = regexp.MustCompile(`(?m)^#import\s+([a-zA-Z][a-zA-Z0-9_]*(?:::[a-zA-Z][a-zA-Z0-9_]*)*)\s*$`)

func replace(re *regexp.Regexp, text string, repl func(match []string) string) string {
	return re.ReplaceAllStringFunc(text, func(s string) string {
		match := re.FindStringSubmatch(s)
		return repl(match)
	})
}

func prepareSource(source string) string {
	source = prepareTemplateForImports(source)
	source = prepareTemplateForValues(source)
	return source
}

func prepareTemplateForImports(source string) string {
	// remove module definitions
	source = reModule.ReplaceAllLiteralString(source, "")

	// rewrite to template calls
	return replace(reImport, source, func(match []string) string {
		mod := match[1]

		return fmt.Sprintf(`
			{{- if importGuard .Imports %[1]q }}
			{{- template %[1]q . }}
			{{- end }}
		`, mod)
	})
}

func prepareTemplateForValues(text string) string {
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

	return text
}

var funcs = template.FuncMap{
	"ifdef": func(values Values, cond string) bool {
		_, ok := values[cond]
		return ok
	},
	"ifndef": func(values Values, cond string) bool {
		_, ok := values[cond]
		return !ok
	},
	"importGuard": func(imported *set.Set[string], name string) bool {
		return imported.Insert(name)
	},
}
