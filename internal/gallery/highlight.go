package main

import (
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"html/template"
	"strings"
)

// classes reuse the highlight.js names, so the github-dark theme styles them.
const (
	classKeyword  = "hljs-keyword"
	classString   = "hljs-string"
	classComment  = "hljs-comment"
	classNumber   = "hljs-number"
	classLiteral  = "hljs-literal"
	classType     = "hljs-type"
	classFunction = "hljs-title function_"
)

var predeclaredTypes = map[string]bool{
	"any": true, "bool": true, "byte": true, "comparable": true, "complex64": true, "complex128": true,
	"error": true, "float32": true, "float64": true, "int": true, "int8": true, "int16": true,
	"int32": true, "int64": true, "rune": true, "string": true, "uint": true, "uint8": true,
	"uint16": true, "uint32": true, "uint64": true, "uintptr": true,
}

var predeclaredLiterals = map[string]bool{"true": true, "false": true, "nil": true, "iota": true}

// highlightGo renders Go source as html. It uses the syntax tree to tell
// types and function names apart from other identifiers.
func highlightGo(src []byte) template.HTML {
	fset := token.NewFileSet()

	// a file that does not parse still gets its tokens highlighted
	file, _ := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)

	// identifier classes by byte offset
	idents := map[int]string{}
	if file != nil {
		classifyIdents(fset, file, idents)
	}

	var s scanner.Scanner
	s.Init(fset.AddFile("", fset.Base(), len(src)), src, nil, scanner.ScanComments)

	var out strings.Builder
	offset := 0

	for {
		pos, tok, lit := s.Scan()
		if tok == token.EOF {
			break
		}

		// skip semicolons inserted at line ends
		if tok == token.SEMICOLON && lit != ";" {
			continue
		}

		start := fset.Position(pos).Offset
		end := start + len(tok.String())
		if lit != "" {
			end = start + len(lit)
		}

		class := ""
		switch {
		case tok == token.COMMENT:
			class = classComment
		case tok == token.STRING || tok == token.CHAR:
			class = classString
		case tok == token.INT || tok == token.FLOAT || tok == token.IMAG:
			class = classNumber
		case tok.IsKeyword():
			class = classKeyword
		case tok == token.IDENT && predeclaredTypes[lit]:
			class = classType
		case tok == token.IDENT && predeclaredLiterals[lit]:
			class = classLiteral
		case tok == token.IDENT:
			class = idents[start]
		}

		if class == "" {
			continue
		}

		out.WriteString(template.HTMLEscapeString(string(src[offset:start])))
		out.WriteString(`<span class="` + class + `">`)
		out.WriteString(template.HTMLEscapeString(string(src[start:end])))
		out.WriteString(`</span>`)
		offset = end
	}

	out.WriteString(template.HTMLEscapeString(string(src[offset:])))

	return template.HTML(out.String())
}

func classifyIdents(fset *token.FileSet, file *ast.File, idents map[int]string) {
	set := func(ident *ast.Ident, class string) {
		idents[fset.Position(ident.Pos()).Offset] = class
	}

	mark := func(e ast.Expr, class string) {
		switch e := e.(type) {
		case *ast.Ident:
			set(e, class)
		case *ast.SelectorExpr:
			set(e.Sel, class)
		}
	}

	var markType func(e ast.Expr)

	markFields := func(fields *ast.FieldList) {
		if fields == nil {
			return
		}

		for _, field := range fields.List {
			markType(field.Type)
		}
	}

	markType = func(e ast.Expr) {
		switch e := e.(type) {
		case *ast.Ident, *ast.SelectorExpr:
			mark(e, classType)
		case *ast.StarExpr:
			markType(e.X)
		case *ast.ParenExpr:
			markType(e.X)
		case *ast.Ellipsis:
			markType(e.Elt)
		case *ast.ArrayType:
			markType(e.Elt)
		case *ast.MapType:
			markType(e.Key)
			markType(e.Value)
		case *ast.ChanType:
			markType(e.Value)
		case *ast.IndexExpr:
			markType(e.X)
			markType(e.Index)
		case *ast.IndexListExpr:
			markType(e.X)
			for _, index := range e.Indices {
				markType(index)
			}
		case *ast.FuncType:
			markFields(e.TypeParams)
			markFields(e.Params)
			markFields(e.Results)
		case *ast.StructType:
			markFields(e.Fields)
		case *ast.InterfaceType:
			for _, method := range e.Methods.List {
				if _, ok := method.Type.(*ast.FuncType); ok && len(method.Names) > 0 {
					set(method.Names[0], classFunction)
				}

				markType(method.Type)
			}
		}
	}

	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncDecl:
			set(node.Name, classFunction)
			markFields(node.Recv)
			markType(node.Type)

		case *ast.FuncLit:
			markType(node.Type)

		case *ast.TypeSpec:
			set(node.Name, classType)
			markFields(node.TypeParams)
			markType(node.Type)

		case *ast.ValueSpec:
			markType(node.Type)

		case *ast.CompositeLit:
			markType(node.Type)

		case *ast.TypeAssertExpr:
			markType(node.Type)

		case *ast.CallExpr:
			switch fun := node.Fun.(type) {
			case *ast.Ident, *ast.SelectorExpr:
				mark(fun, classFunction)
			case *ast.IndexExpr:
				// generic call like Spawn[T](...)
				mark(fun.X, classFunction)
				markType(fun.Index)
			case *ast.IndexListExpr:
				mark(fun.X, classFunction)
				for _, index := range fun.Indices {
					markType(index)
				}
			default:
				// conversion like []byte(x)
				markType(fun)
			}
		}

		return true
	})
}
