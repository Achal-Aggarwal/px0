package fuzzy

import (
	"fmt"
	"strings"
	"testing"
)

func subsequence(sub, s string) bool {
	si := 0
	for i := 0; i < len(s) && si < len(sub); i++ {
		if s[i] == sub[si] {
			si++
		}
	}
	return si == len(sub)
}

func TestFuzzyRanking(t *testing.T) {
	t.Parallel()

	paths := []string{
		"internal/server/http_server.go",
		"cmd/px0/main.go",
		"web/app.js",
		"pkg/util/strings.go",
		"vendor/github.com/x/http/server.go",
		"httpserver.go",
	}
	items := make([]Item, len(paths))
	for i, p := range paths {
		items[i] = NewItem(p)
	}

	got := Find(items, "httpserver", 10)
	if len(got) == 0 {
		t.Fatal("no matches for httpserver")
	}
	if got[0].Path != "httpserver.go" {
		t.Errorf("best match for httpserver = %q, want httpserver.go", got[0].Path)
	}

	for _, r := range got {
		if !subsequence("httpserver", strings.ToLower(r.Path)) {
			t.Errorf("%q is not a subsequence match", r.Path)
		}
		for _, p := range r.Pos {
			if p < 0 || p >= len(r.Path) {
				t.Errorf("%q: highlight position %d out of range", r.Path, p)
			}
		}
	}

	if got := Find(items, "zzqq", 10); len(got) != 0 {
		t.Errorf("expected no matches, got %v", got)
	}

	if got := Find(items, "appjs", 10); len(got) == 0 || got[0].Path != "web/app.js" {
		t.Errorf("expected web/app.js for appjs, got %v", got)
	}
}

func TestFuzzyCaseSensitivity(t *testing.T) {
	t.Parallel()

	items := []Item{
		NewItem("src/HTTPServer.go"),
		NewItem("src/httpserver.go"),
	}

	res := Find(items, "HTTPServer", 10)
	if len(res) < 2 {
		t.Fatalf("expected 2 results, got %d", len(res))
	}
	if res[0].Path != "src/HTTPServer.go" {
		t.Errorf("expected 'src/HTTPServer.go' to rank higher for query 'HTTPServer', got: %s", res[0].Path)
	}
}

func TestFuzzyEmptyQuery(t *testing.T) {
	t.Parallel()

	items := []Item{
		NewItem("a.go"),
		NewItem("b.go"),
		NewItem("c.go"),
	}
	got := Find(items, "", 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 items for empty query limit 2, got %d", len(got))
	}
	if got[0].Path != "a.go" || got[1].Path != "b.go" {
		t.Errorf("unexpected results: %+v", got)
	}
}

func TestIsBoundary(t *testing.T) {
	t.Parallel()

	boundaries := []byte{'/', '_', '-', '.', ' ', '@'}
	for _, b := range boundaries {
		if !IsBoundary(b) {
			t.Errorf("expected byte %q to be boundary", b)
		}
	}
	nonBoundaries := []byte{'a', 'Z', '0', '!', ':'}
	for _, nb := range nonBoundaries {
		if IsBoundary(nb) {
			t.Errorf("expected byte %q not to be boundary", nb)
		}
	}
}

func TestFuzzyLimitAndHeap(t *testing.T) {
	t.Parallel()

	paths := []string{
		"a/b/c/server.go",
		"server.go",
		"src/server.go",
		"pkg/net/server.go",
		"cmd/server/server.go",
		"internal/server.go",
		"vendor/server.go",
	}
	items := make([]Item, len(paths))
	for i, p := range paths {
		items[i] = NewItem(p)
	}

	// Request limit of 3 when 7 matches exist - tests min-heap, worseResult and siftDownResult
	res := Find(items, "server", 3)
	if len(res) != 3 {
		t.Fatalf("expected 3 results with limit 3, got %d", len(res))
	}
	// Best match should be top level server.go
	if res[0].Path != "server.go" {
		t.Errorf("expected top result 'server.go', got %q", res[0].Path)
	}

	// Test no limit (limit <= 0)
	all := Find(items, "server", 0)
	if len(all) != len(paths) {
		t.Fatalf("expected %d results with limit 0, got %d", len(paths), len(all))
	}
}

func BenchmarkFuzzyFind(b *testing.B) {
	paths := []string{
		"internal/server/http_server.go",
		"cmd/px0/main.go",
		"web/app.js",
		"pkg/util/strings.go",
		"vendor/github.com/x/http/server.go",
		"httpserver.go",
		"src/core/router.go",
		"api/v1/endpoints.go",
	}
	items := make([]Item, len(paths))
	for i, p := range paths {
		items[i] = NewItem(p)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Find(items, "httpserver", 5)
	}
}

func BenchmarkScore(b *testing.B) {
	item := NewItem("internal/server/http_server.go")
	scratch := make([]int, 0, 64)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _, _ = Score("httpserver", "httpserver", item.Path, item.Lower, item.NameStart, scratch)
	}
}

func generateBenchItems(n int) []Item {
	dirs := []string{"src", "pkg", "internal", "cmd", "pkg/util", "web/src/components", "api/v1", "services"}
	names := []string{"server.go", "client.go", "handler.go", "types.go", "utils.go", "index.ts", "app.py", "main.rs"}
	entries := make([]Item, n)
	for i := 0; i < n; i++ {
		d := dirs[i%len(dirs)]
		name := fmt.Sprintf("%d_%s", i, names[i%len(names)])
		p := fmt.Sprintf("%s/%s", d, name)
		entries[i] = Item{
			Path:      p,
			Name:      name,
			Lower:     strings.ToLower(p),
			NameStart: len(p) - len(name),
		}
	}
	return entries
}

func BenchmarkFuzzyFind_1K(b *testing.B) {
	files := generateBenchItems(1000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Find(files, "handler", 50)
	}
}

func BenchmarkFuzzyFind_10K(b *testing.B) {
	files := generateBenchItems(10000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Find(files, "handler", 50)
	}
}

func BenchmarkFuzzyFind_50K(b *testing.B) {
	files := generateBenchItems(50000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Find(files, "srvutil", 50)
	}
}

func BenchmarkFuzzyFindShortQuery_50K(b *testing.B) {
	files := generateBenchItems(50000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Find(files, "in", 50)
	}
}

