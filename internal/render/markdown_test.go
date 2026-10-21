package render

import "testing"

func TestHeadingIDsFollowGitHub(t *testing.T) {
	t.Parallel()
	ids := &headingIDs{seen: map[string]bool{}}
	for _, c := range []struct{ in, want string }{
		{"Getting Started!", "getting-started"},
		{"snake_case API", "snake_case-api"},
		{"Überblick", "überblick"},
		{"Getting Started", "getting-started-1"},
		{"???", "section"},
	} {
		if got := string(ids.Generate([]byte(c.in), 0)); got != c.want {
			t.Errorf("%q: got %q, want %q", c.in, got, c.want)
		}
	}
}

func TestRenderMarkdown(t *testing.T) {
	t.Parallel()

	if !IsMarkdown("doc.md") || !IsMarkdown("doc.markdown") || IsMarkdown("code.go") {
		t.Errorf("IsMarkdown incorrect results")
	}

	src := []byte("# Header\n\nParagraph with **bold** and `code`.\n\n```go\nfunc main() {}\n```\n")
	html, err := RenderMarkdown(src)
	if err != nil {
		t.Fatalf("RenderMarkdown returned error: %v", err)
	}
	if len(html) == 0 {
		t.Fatal("expected non-empty rendered markdown")
	}
}

func BenchmarkRenderMarkdown(b *testing.B) {
	src := []byte("# Header\n\nParagraph with **bold** text and list:\n- item 1\n- item 2\n\n```go\npackage main\nfunc main() {}\n```\n")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = RenderMarkdown(src)
	}
}

