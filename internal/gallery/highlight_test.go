package main

import (
	"html"
	"regexp"
	"strings"
	"testing"
)

func TestHighlightGo(t *testing.T) {
	src := `package main

// Speed of a <thing>
type Speed struct{ Value float32 }

func move(q byke.Query[struct{ Speed *Speed }]) {
	x := []byte("a&b")
	byke.Spawn[Speed](nil, 1.5)
}
`

	out := string(highlightGo([]byte(src)))

	// removing the markup must give back the source
	plain := html.UnescapeString(regexp.MustCompile(`<[^>]+>`).ReplaceAllString(out, ""))
	if plain != src {
		t.Fatalf("source changed:\n%s", plain)
	}

	for _, want := range []string{
		`<span class="hljs-keyword">package</span>`,
		`<span class="hljs-comment">// Speed of a &lt;thing&gt;</span>`,
		`<span class="hljs-type">Speed</span> <span class="hljs-keyword">struct</span>`,
		`<span class="hljs-type">float32</span>`,
		`<span class="hljs-title function_">move</span>`,
		`byke.<span class="hljs-type">Query</span>`,
		`*<span class="hljs-type">Speed</span>`,
		`<span class="hljs-string">&#34;a&amp;b&#34;</span>`,
		`byke.<span class="hljs-title function_">Spawn</span>[<span class="hljs-type">Speed</span>]`,
		`<span class="hljs-literal">nil</span>`,
		`<span class="hljs-number">1.5</span>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in:\n%s", want, out)
		}
	}
}
