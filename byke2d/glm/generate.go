//go:build generate

package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"strings"
	"text/template"
)

//go:generate go run generate.go

func main() {
	generateVecs()
	generateMats()
	generateRects()
}

func generateVecs() {
	var vecTypes []modelVec

	for _, ty := range []string{"float32", "float64", "uint32", "uint16"} {
		vecTypes = append(vecTypes,
			modelVec{
				Len:           2,
				LenWGPU:       2,
				ComponentType: ty,
			},
			modelVec{
				Len:           3,
				LenWGPU:       4,
				ComponentType: ty,
			},
			modelVec{
				Len:           4,
				LenWGPU:       4,
				ComponentType: ty,
			},
		)
	}

	for _, vecType := range vecTypes {

		var buf bytes.Buffer
		err := tmplVec.Execute(&buf, vecType)

		if err != nil {
			panic(err)
		}

		suffix := componentTypeToSuffix(vecType.ComponentType)
		writeCode(fmt.Sprintf("gen_vec%d%s.go", vecType.Len, suffix), buf.String())
	}
}

func generateMats() {
	matTypes := []modelMat{}

	for _, ty := range []string{"float32", "float64", "uint32", "uint16"} {
		matTypes = append(matTypes,

			modelMat{
				Len:           2,
				ComponentType: ty,
			},
			modelMat{
				Len:           3,
				ComponentType: ty,
			},
			modelMat{
				Len:           4,
				ComponentType: ty,
			},
		)
	}

	for _, matType := range matTypes {
		var buf bytes.Buffer
		err := tmplMat.Execute(&buf, matType)

		if err != nil {
			panic(err)
		}

		suffix := componentTypeToSuffix(matType.ComponentType)
		writeCode(fmt.Sprintf("gen_mat%d%s.go", matType.Len, suffix), buf.String())
	}
}

func generateRects() {
	rectTypes := []modelRect{}

	for _, ty := range []string{"float32", "float64", "uint32", "uint16"} {
		rectTypes = append(rectTypes,
			modelRect{
				ComponentType: ty,
			},
		)
	}

	for _, rectType := range rectTypes {
		var buf bytes.Buffer
		err := tmplRect.Execute(&buf, rectType)

		if err != nil {
			panic(err)
		}

		suffix := componentTypeToSuffix(rectType.ComponentType)
		writeCode(fmt.Sprintf("gen_rect%s.go", suffix), buf.String())
	}
}

func writeCode(name string, source string) {
	formatted, err := format.Source([]byte(source))
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(name, formatted, 0644)
	if err != nil {
		panic(err)
	}
}

type Swizzle struct {
	Name       string
	Components []int
}

func generateSwizzles(n int) []Swizzle {
	var swizzles []Swizzle
	for _, swizzle := range generateAllSwizzles(n, n) {
		if len(swizzle.Components) <= 1 {
			continue
		}

		swizzles = append(swizzles, swizzle)
	}

	return swizzles
}

func generateAllSwizzles(lenVec, n int) []Swizzle {
	if n == 0 {
		return []Swizzle{
			// empty swizzle
			{Components: []int{}},
		}
	}

	swizzles := generateAllSwizzles(lenVec, n-1)

	for _, swizzle := range swizzles {
		if len(swizzle.Components) != n-1 {
			continue
		}

		for idx := range lenVec {
			components := append([]int(nil), swizzle.Components...)

			swizzles = append(swizzles, Swizzle{
				Name:       fmt.Sprintf("%s%c", swizzle.Name, "XYZW"[idx]),
				Components: append(components, idx),
			})
		}
	}

	return swizzles

}

func componentTypeToSuffix(ty string) string {
	switch ty {
	case "float32":
		return "f"
	case "float64":
		return "d"
	case "uint32":
		return "u"
	case "uint16":
		return "uh"
	default:
		panic("Unsupported component type: " + ty)
	}
}

type modelVec struct {
	Len           int
	LenWGPU       int
	ComponentType string
}

func (s modelVec) ComponentTypeSuffix() string {
	return componentTypeToSuffix(s.ComponentType)
}

func (s modelVec) Swizzles() []Swizzle {
	return generateSwizzles(s.Len)
}

func (s modelVec) Name() string {
	return s.NameWithSuffix(componentTypeToSuffix(s.ComponentType))
}

func (s modelVec) NameWithSuffix(suffix string) string {
	return fmt.Sprintf("Vec%d%s", s.Len, suffix)
}

func (s modelVec) Type() string {
	return s.Name()
}

type modelMat struct {
	Len           int
	ComponentType string
}

func (s modelMat) Name() string {
	return fmt.Sprintf("Mat%d%s", s.Len, componentTypeToSuffix(s.ComponentType))
}

func (s modelMat) Type() string {
	return s.Name()
}

func (s modelMat) ComponentTypeSuffix() string {
	return componentTypeToSuffix(s.ComponentType)
}

func (s modelMat) IsFloat() bool {
	return strings.HasPrefix(s.ComponentType, "float")
}

func (s modelMat) ValueCount() int {
	return s.Len * s.Len
}

func (s modelMat) ArrayType() string {
	return fmt.Sprintf("[%d][%d]%s", s.Len, s.Len, s.ComponentType)
}

func (s modelMat) At(name string, x, y int) string {
	var res = name

	res += fmt.Sprintf("[%d][%d]", y, x)
	if x == y {
		res = fmt.Sprintf("(%s+1)", res)
	}

	return res
}

type modelRect struct {
	ComponentType string
}

func (s modelRect) Type() string {
	return fmt.Sprintf("Rect%s", s.ComponentTypeSuffix())
}

func (s modelRect) ComponentTypeSuffix() string {
	return componentTypeToSuffix(s.ComponentType)
}

func (s modelRect) VecType() string {
	return fmt.Sprintf("Vec2%s", s.ComponentTypeSuffix())
}

var funcs = template.FuncMap{
	"comma": func(idx int) string {
		if idx == 0 {
			return " "
		}
		return ","

	},
	"sep": func(idx int, empty, sep string) string {
		if idx == 0 {
			return empty
		}
		return sep

	},
	"plus": func(a, b int) int {
		return a + b
	},
	"minus": func(a, b int) int {
		return a - b
	},
	"component": func(idx int) string {
		names := []string{"x", "y", "z", "w"}
		return names[idx]
	},
	"toComponents": func(len int) string {
		return "XYZW"[:len]
	},
}

var tmplVec = template.Must(template.New("").Funcs(funcs).Parse(`
// Code generated by generate.go: DO NOT EDIT.

package glm

import "fmt"
import "math"

// {{.Name}} is a vector of dimension {{.Len}}.
type {{.Name}} [{{ .Len }}]{{.ComponentType}}

func (lhs {{.Type}}) Dot(rhs {{.Type}}) {{.ComponentType}} {
	return {{ range $idx := .Len }} {{ sep $idx "" "+"}} lhs[{{$idx}}] * rhs[{{$idx}}] {{ end }}
}

func (lhs {{.Type}}) LengthSqr() {{.ComponentType}} {
	return lhs.Dot(lhs)
}

func (lhs {{.Type}}) Length() {{.ComponentType}} {
	return {{.ComponentType}}(math.Sqrt(float64(lhs.Dot(lhs))))
}

func (lhs {{.Type}}) Normalize() {{.Type}} {
	return lhs.Scale(1.0 / lhs.Length())
}

func (lhs {{.Type}}) Scale(s {{.ComponentType}}) {{.Type}} {
	return {{.Type}}{
		{{- range $idx := .Len }}
		lhs[{{$idx}}] * s,
		{{- end }}
	}
}

func (lhs {{.Type}}) Reciprocal() {{.Type}} {
	return {{.Type}}{
		{{- range $idx := .Len }}
		1.0 / lhs[{{$idx}}],
		{{- end }}
	}
}

func (lhs {{.Type}}) Add(rhs {{.Type}}) {{.Type}} {
	return {{.Type}}{
		{{- range $idx := .Len }}
		lhs[{{$idx}}] + rhs[{{$idx}}],
		{{- end }}
	}
}

func (lhs {{.Type}}) Sub(rhs {{.Type}}) {{.Type}} {
	return {{.Type}}{
		{{- range $idx := .Len }}
		lhs[{{$idx}}] - rhs[{{$idx}}],
		{{- end }}
	}
}

func (lhs {{.Type}}) Mul(rhs {{.Type}}) {{.Type}} {
	return {{.Type}}{
		{{- range $idx := .Len }}
		lhs[{{$idx}}] * rhs[{{$idx}}],
		{{- end }}
	}
}

func (lhs {{.Type}}) Div(rhs {{.Type}}) {{.Type}} {
	return {{.Type}}{
		{{- range $idx := .Len }}
		lhs[{{$idx}}] / rhs[{{$idx}}],
		{{- end }}
	}
}

func (lhs {{.Type}}) Min(rhs {{.Type}}) {{.Type}} {
	return {{.Type}}{
		{{- range $idx := .Len }}
		min(lhs[{{$idx}}], rhs[{{$idx}}]),
		{{- end }}
	}
}

func (lhs {{.Type}}) Max(rhs {{.Type}}) {{.Type}} {
	return {{.Type}}{
		{{- range $idx := .Len }}
		max(lhs[{{$idx}}], rhs[{{$idx}}]),
		{{- end }}
	}
}

{{ if (lt $.Len 4) }}
	func (lhs {{.Type}}) Extend({{component .Len}} {{.ComponentType}}) Vec{{plus .Len 1}}{{.ComponentTypeSuffix}} {
		return Vec{{plus .Len 1}}{{.ComponentTypeSuffix}}{
			{{- range $idx := .Len }}
			lhs[{{$idx}}],
			{{- end }}
			{{component .Len}},
		}
}
{{ end  }}

{{ if (gt $.Len 2) }}
	func (lhs {{.Type}}) Truncate() Vec{{minus .Len 1}}{{.ComponentTypeSuffix}} {
		return Vec{{minus .Len 1}}{{.ComponentTypeSuffix}}{
			{{- range $idx := (minus .Len 1) }}
			lhs[{{$idx}}],
			{{- end }}
		}
	}
{{ end }}

{{ range $swizzle := .Swizzles }}
	// Swizzle{{ $swizzle.Name }} returns a new vector with the components of this
	// vector in the order of {{ $swizzle.Name }}.
	func (lhs {{$.Type}}) Swizzle{{$swizzle.Name}}() Vec{{ $swizzle.Components|len }}{{$.ComponentTypeSuffix}} {
		return Vec{{ $swizzle.Components|len }}{{$.ComponentTypeSuffix}}{
			{{- range $idx := $swizzle.Components }}
			lhs[{{$idx}}],
			{{- end }}
		}
	}
{{ end }}

{{ range $len := (plus .Len 1) }}
	{{ if (gt $len 1) }}
	func (lhs {{$.Type}}) {{toComponents $len}}() ({{ range $idx := $len}}{{comma $idx}}{{ component $idx }}{{end}} {{$.ComponentType}}) {
		{{- range $idx := $len }}
		{{component $idx}} = lhs[{{$idx}}]
		{{- end }}
		return
	}
	{{ end }}
{{ end }}

func (lhs {{.Type}}) To{{ .NameWithSuffix "f" }}() {{ .NameWithSuffix "f" }} {
	return {{ .NameWithSuffix "f" }}{
		{{- range $idx := .Len }}
		float32(lhs[{{$idx}}]),
		{{- end }}
	}
}

func (lhs {{.Type}}) ToWGPU() [{{.LenWGPU}}]float32 {
	return [{{.LenWGPU}}]float32{
		{{- range $idx := .Len }}
		float32(lhs[{{$idx}}]),
		{{- end }}
	}
}

func (lhs {{.Type}}) String() string {
	return fmt.Sprintf(
		"vec{{.Len}}({{range $idx := .Len}}{{sep $idx "" ", "}}%v{{end}})",
		{{- range $idx := .Len }}
		lhs[{{$idx}}],
		{{- end }}
	)
}

`))

var tmplMat = template.Must(template.New("").Funcs(funcs).Parse(`
// Code generated by generate.go: DO NOT EDIT.

package glm

// {{.Name}} is a {{.Len}}x{{.Len}} matrix.
// The default value is the identity matrix.
type {{.Name}} struct {
	values {{.ArrayType}}
}

func {{.Name}}Of(values {{.ArrayType}}) {{.Type}} {
	return {{ .Type }}{
		values: {{.ArrayType}}{
		{{- range $y := $.Len }}
			{
			{{- range $x := $.Len }}
				values[{{$y}}][{{$x}}] {{if eq $x $y}}-1{{end}},
			{{- end }}
			},
		{{- end }}
		},
	}
}

func Identity{{.Name}}() {{.Type}} {
	return {{.Type}}{}
}

func (m {{.Type}}) Mul(o {{.Type}}) {{.Type}} {
	mv := &m.values
	ov := &o.values

	return {{ .Type }}{
		values: {{.ArrayType}}{
		{{- range $y := $.Len }}
			{
			{{ range $x := $.Len }}
				{{- range $i := $.Len -}}
					{{ sep $i " " "+" -}}
					{{ $.At "mv" $x $i }} * {{ $.At "ov" $i $y }}
				{{- end -}} {{ if eq $x $y }} -1 {{ end }},
			{{ end }}
			},
		{{- end }}
		},
	}
}

func (m {{.Type}}) Transpose() {{.Type}} {
	mv := &m.values

	return {{ .Type }}{
		values: {{ .ArrayType }}{
		{{- range $y := $.Len }}
			{
			{{- range $x := $.Len }}
				mv[{{$x}}][{{$y}}],
			{{- end }}
			},
		{{- end }}
		},
	}
}

func (m {{.Type}}) Components() {{.ArrayType}} {
	values := m.values
	{{- range $i := $.Len }}
	values[{{$i}}][{{$i}}] += 1
	{{- end }}
	return values
}

{{ range $y := $.Len }}
{{ range $x := $.Len }}
	func (m *{{$.Type}}) m{{$y}}{{$x}}() {{$.ComponentType}} {
		return {{ $.At "m.values" $x $y }}
	}
{{ end }}
{{ end }}

{{ if eq .Len 3 }} 
	func Translation{{ .Name }}(x, y {{.ComponentType}}) {{.Type}} {
		return {{ .Name }}Of([3][3]{{ .ComponentType }}{
			{1, 0, 0},
			{0, 1, 0},
			{x, y, 1},
		})
	}
	
	func Scale{{ .Name }}(x, y {{.ComponentType}}) {{.Type}} {
		return {{ .Name }}Of([3][3]{{.ComponentType}}{
			{x, 0, 0},
			{0, y, 0},
			{0, 0, 1},
		})
	}
	
	func (m {{.Type}}) Translate(x, y {{.ComponentType}}) {{.Type}} {
		return m.Mul(Translation{{.Type}}(x, y))
	}
	
	func (m {{.Type}}) Scale(x, y {{.ComponentType}}) {{.Type}} {
		return m.Mul(Scale{{.Type}}(x, y))
	}
	
	{{ if .IsFloat }}
		func Rotation{{ .Name }}(angle Rad) {{.Type}} {
			s, c := fastSincos(angle)
		
			return {{ .Name }}Of([3][3]{{.ComponentType}}{
				{ {{.ComponentType}}(c), {{.ComponentType}}(s), 0},
				{-{{.ComponentType}}(s), {{.ComponentType}}(c), 0},
				{0, 0, 1},
			})
		}
	
		func (m {{.Type}}) Rotate(angle Rad) {{.Type}} {
				return m.Mul(Rotation{{.Type}}(angle))
		}
	{{ end }}

	func (m {{.Type}}) Row(i int) Vec3{{.ComponentTypeSuffix}} {
		if i == 0 {
			return Vec3{{.ComponentTypeSuffix}}{
				m.m00(),
				m.m10(),
				m.m20(),
			}
		}
		if i == 1 {
			return Vec3{{.ComponentTypeSuffix}}{m.m01(), m.m11(), m.m21()}
		}
		if i == 2 {
			return Vec3{{.ComponentTypeSuffix}}{
				m.m02(),
				m.m12(),
				m.m22(),
			}
		}
	
		panic(i)
	}
	
	func (m {{.Type}}) ToWGPU() [12]float32 {
		return [12]float32{
			float32(m.m00()), float32(m.m01()), float32(m.m02()), 0,
			float32(m.m10()), float32(m.m11()), float32(m.m12()), 0,
			float32(m.m20()), float32(m.m21()), float32(m.m22()), 0,
		}
	}
	
	{{ if .IsFloat }}
		func (m {{.Type}}) Invert() {{.Type}} {
			inv, ok := m.TryInvert()
			if !ok {
				panic("matrix not invertible")
			}
		
			return inv
		}
	
		func (m {{.Type}}) TryInvert() ({{.Type}}, bool) {
			var inv [3][3]{{.ComponentType}}
		
			// determinant
			det := m.m00()*(m.m11()*m.m22()-m.m12()*m.m21()) -
				m.m01()*(m.m10()*m.m22()-m.m12()*m.m20()) +
				m.m02()*(m.m10()*m.m21()-m.m11()*m.m20())
		
			if det == 0 {
				// singular
				return {{.Type}}{}, false
			}
		
			inv[0][0] = (m.m11()*m.m22() - m.m12()*m.m21()) / det
			inv[0][1] = (m.m02()*m.m21() - m.m01()*m.m22()) / det
			inv[0][2] = (m.m01()*m.m12() - m.m02()*m.m11()) / det
		
			inv[1][0] = (m.m12()*m.m20() - m.m10()*m.m22()) / det
			inv[1][1] = (m.m00()*m.m22() - m.m02()*m.m20()) / det
			inv[1][2] = (m.m02()*m.m10() - m.m00()*m.m12()) / det
		
			inv[2][0] = (m.m10()*m.m21() - m.m11()*m.m20()) / det
			inv[2][1] = (m.m01()*m.m20() - m.m00()*m.m21()) / det
			inv[2][2] = (m.m00()*m.m11() - m.m01()*m.m10()) / det
		
			return {{.Name}}Of(inv), true
		}
	{{ end }}	

	func (m {{.Type}}) Transform(vec Vec3{{.ComponentTypeSuffix}}) Vec3{{.ComponentTypeSuffix}} {
		return Vec3{{.ComponentTypeSuffix}}{
			(m.m00())*vec[0] + (m.m10())*vec[1] + (m.m20())*vec[2],
			(m.m01())*vec[0] + (m.m11())*vec[1] + (m.m21())*vec[2],
			(m.m02())*vec[0] + (m.m12())*vec[1] + (m.m22())*vec[2],
		}
	}
	
	func (m {{.Type}}) Transform2(vec Vec2{{.ComponentTypeSuffix}}) Vec2{{.ComponentTypeSuffix}} {
		return Vec2{{.ComponentTypeSuffix}}{
			(m.m00())*vec[0] + (m.m10())*vec[1] + (m.m20()),
			(m.m01())*vec[0] + (m.m11())*vec[1] + (m.m21()),
		}
	}
{{ end }}

{{ if eq .Len 4 }}
	func Translation{{.Name}}(x, y, z {{.ComponentType}}) {{.Type}} {
		return {{.Type}}{
			values: [4][4]{{.ComponentType}}{
				{0, 0, 0, 0},
				{0, 0, 0, 0},
				{0, 0, 0, 0},
				{x, y, z, 0},
			},
		}
	}
	
	func Scale{{.Name}}(x, y, z {{.ComponentType}}) {{.Type}} {
		x -= 1
		y -= 1
		z -= 1
	
		return {{.Type}}{
			values: [4][4]{{.ComponentType}}{
				{x, 0, 0, 0},
				{0, y, 0, 0},
				{0, 0, z, 0},
				{0, 0, 0, 0},
			},
		}
	}
	
	func (m {{.Type}}) Translate(x, y, z {{.ComponentType}}) {{.Type}} {
		return m.Mul(Translation{{.Name}}(x, y, z))
	}
	
	func (m {{.Type}}) Scale(x, y, z {{.ComponentType}}) {{.Type}} {
		return m.Mul(Scale{{.Name}}(x, y, z))
	}
	
	{{ if .IsFloat }}
		func RotationZ{{.Name}}(angle Rad) {{.Type}} {
			fs, fc := fastSincos(angle)
			s := {{.ComponentType}}(fs)
			c := {{.ComponentType}}(fc) - 1
		
			return {{.Type}}{
				values: [4][4]{{.ComponentType}}{
					{c, s, 0, 0},
					{-s, c, 0, 0},
					{0, 0, 0, 0},
					{0, 0, 0, 0},
				},
			}
		}
		
		func RotationX{{.Name}}(angle Rad) {{.Type}} {
			fs, fc := fastSincos(angle)
			s := {{.ComponentType}}(fs)
			c := {{.ComponentType}}(fc) - 1
		
			return {{.Type}}{
				values: [4][4]{{.ComponentType}}{
					{0, 0, 0, 0},
					{0, c, s, 0},
					{0, -s, c, 0},
					{0, 0, 0, 0},
				},
			}
		}
		
		func RotationY{{.Name}}(angle Rad) {{.Type}} {
			fs, fc := fastSincos(angle)
			s := {{.ComponentType}}(fs)
			c := {{.ComponentType}}(fc) - 1
		
			return {{.Type}}{
				values: [4][4]{{.ComponentType}}{
					{c, 0, s, 0},
					{0, 0, 0, 0},
					{-s, 0, c, 0},
					{0, 0, 0, 0},
				},
			}
		}
		

		func (m {{.Type}}) RotateX(angle Rad) {{.Type}} {
			return m.Mul(RotationX{{.Name}}(angle))
		}
		
		func (m {{.Type}}) RotateY(angle Rad) {{.Type}} {
			return m.Mul(RotationY{{.Name}}(angle))
		}
		
		func (m {{.Type}}) RotateZ(angle Rad) {{.Type}} {
			return m.Mul(RotationZ{{.Name}}(angle))
		}
	{{ end }}
	
	func (m {{.Type}}) Transform(vec Vec4{{.ComponentTypeSuffix}}) Vec4{{.ComponentTypeSuffix}} {
		return Vec4{{.ComponentTypeSuffix}}{
			(m.m00())*vec[0] + (m.m10())*vec[1] + (m.m20())*vec[2] + (m.m30())*vec[3],
			(m.m01())*vec[0] + (m.m11())*vec[1] + (m.m21())*vec[2] + (m.m31())*vec[3],
			(m.m02())*vec[0] + (m.m12())*vec[1] + (m.m22())*vec[2] + (m.m32())*vec[3],
			(m.m03())*vec[0] + (m.m13())*vec[1] + (m.m23())*vec[2] + (m.m33())*vec[3],
		}
	}
	
	func (m {{.Type}}) Transform3(vec Vec3{{.ComponentTypeSuffix}}) Vec3{{.ComponentTypeSuffix}} {
		return Vec3{{.ComponentTypeSuffix}}{
			(m.m00())*vec[0] + (m.m10())*vec[1] + (m.m20() * vec[2]) + m.m30(),
			(m.m01())*vec[0] + (m.m11())*vec[1] + (m.m21() * vec[2]) + m.m31(),
			(m.m02())*vec[0] + (m.m12())*vec[1] + (m.m22() * vec[2]) + m.m32(),
		}
	}
	
	func (m {{.Type}}) Transform2(vec Vec2{{.ComponentTypeSuffix}}) Vec2{{.ComponentTypeSuffix}} {
		return Vec2{{.ComponentTypeSuffix}}{
			(m.m00())*vec[0] + (m.m10())*vec[1] + m.m30(),
			(m.m01())*vec[0] + (m.m11())*vec[1] + m.m31(),
		}
	}
	
	func (m {{.Type}}) TranslateZ() {{.ComponentType}} {
		return m.values[3][2]
	}


{{ end }}

`))

var tmplRect = template.Must(template.New("").Funcs(funcs).Parse(`
// Code generated by generate.go: DO NOT EDIT.

package glm

import (
	"fmt"
)

type {{ .Type }} struct {
	Min {{.VecType}}
	Max {{.VecType}}
}

func {{.Type}}FromSize(pos {{.VecType}}, size {{.VecType}}) {{.Type}}  {
	return {{.Type}}FromPoints(pos, pos.Add(size))
}

func {{.Type}}FromXYWH(x, y, w, h {{.ComponentType}}) {{.Type}}  {
	pos := {{.VecType}}{x, y}
	size := {{.VecType}}{w, h}
	return {{.Type}}FromSize(pos, size)
}

func {{.Type}}FromPoints(a, b {{.VecType}}) {{.Type}}  {
	return {{.Type}} {
		Min: {{.VecType}}{
			min(a[0], b[0]),
			min(a[1], b[1]),
		},
		Max: {{.VecType}}{
			max(a[0], b[0]),
			max(a[1], b[1]),
		},
	}
}

func (r {{.Type}} ) Extend(point {{.VecType}}) {{.Type}}  {
	minX := min(r.Min[0], point[0])
	minY := min(r.Min[1], point[1])

	maxX := max(r.Max[0], point[0])
	maxY := max(r.Max[1], point[1])

	return {{.Type}} {
		Min: {{.VecType}}{minX, minY},
		Max: {{.VecType}}{maxX, maxY},
	}
}

func (r {{.Type}} ) Union(other {{.Type}} ) {{.Type}}  {
	return r.Extend(other.Min).Extend(other.Max)
}

func (r {{.Type}} ) Contains(other {{.Type}} ) bool {
	return r.Min[0] <= other.Min[0] && r.Min[1] <= other.Min[1] &&
		r.Max[0] >= other.Max[0] && r.Max[1] >= other.Max[1]
}

func (r {{.Type}} ) Center() {{.VecType}} {
	return r.Min.Add(r.Max).Div({{.VecType}}{2, 2})
}

func (r {{.Type}} ) Offset() {{.VecType}} {
	return r.Min
}

func (r {{.Type}} ) Size() {{.VecType}} {
	return r.Max.Sub(r.Min)
}

func (r {{.Type}} ) Width() {{.ComponentType}} {
	return r.Max[0] - r.Min[0]
}

func (r {{.Type}} ) Height() {{.ComponentType}} {
	return r.Max[1] - r.Min[1]
}

func (r {{.Type}} ) XYWH() ({{.ComponentType}}, {{.ComponentType}}, {{.ComponentType}}, {{.ComponentType}}) {
	x, y := r.Min.XY()
	w, h := r.Size().XY()
	return x, y, w, h
}

func (r {{.Type}} ) String() string {
	x, y, w, h := r.XYWH()
	return fmt.Sprintf("Rect(x=%v, y=%v, w=%v, h=%v)", x, y, w, h)
}

`))
