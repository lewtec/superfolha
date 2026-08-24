package paths

import (
	"strings"
	"testing"
)

func TestBuilders(t *testing.T) {
	t.Parallel()
	if Login() != "/login" || Register() != "/register" || Projects() != "/sessions" {
		t.Fatal("root builders")
	}
	if got := Editor("abc"); got != "/editor/abc" {
		t.Fatalf("Editor = %q", got)
	}
	if got := ProjectDelete("abc"); got != "/sessions/abc/end" {
		t.Fatalf("ProjectDelete = %q", got)
	}
	if got := BrandLogo(); got != "/static/brand/logo.png" {
		t.Fatalf("BrandLogo = %q", got)
	}
	if got := Upload("abc"); got != "/api/projects/abc/upload-file" {
		t.Fatalf("Upload = %q", got)
	}
	if got := ProjectWS("abc"); got != "/ws/projects/abc" {
		t.Fatalf("ProjectWS = %q", got)
	}
}

func TestPatternsMatchBuilders(t *testing.T) {
	t.Parallel()
	cases := []struct {
		pattern string
		sample  string
	}{
		{PatternLanding, Landing},
		{PatternLoginGet, Login()},
		{PatternLoginPost, Login()},
		{PatternRegisterGet, Register()},
		{PatternRegisterPost, Register()},
		{PatternLogout, Logout()},
		{PatternLang, Lang()},
		{PatternProjectsGet, Projects()},
		{PatternProjectsPost, Projects()},
		{PatternProjectDelete, ProjectDelete("x")},
		{PatternEditorGet, Editor("x")},
		{PatternStatic, Static("style.css")},
		{PatternAPILogout, APILogout()},
		{PatternCompile, Compile("x", "main.tex")},
		{PatternUpload, Upload("x")},
		{PatternDownload, Download("x", "main.tex")},
		{PatternProjectWS, ProjectWS("x")},
	}
	for _, tc := range cases {
		pathPart := tc.pattern
		if i := strings.IndexByte(pathPart, ' '); i >= 0 {
			pathPart = pathPart[i+1:]
		}
		sample := tc.sample
		if i := strings.IndexByte(sample, '?'); i >= 0 {
			sample = sample[:i]
		}
		if pathPart == "/{$}" {
			if sample != "/" {
				t.Errorf("pattern %q vs sample %q", tc.pattern, tc.sample)
			}
			continue
		}
		if pathPart == "/static/" {
			if !strings.HasPrefix(sample, "/static/") && sample != "/static" {
				t.Errorf("pattern %q vs sample %q", tc.pattern, tc.sample)
			}
			continue
		}
		patSegs := strings.Split(strings.Trim(pathPart, "/"), "/")
		samSegs := strings.Split(strings.Trim(sample, "/"), "/")
		if len(patSegs) != len(samSegs) {
			t.Errorf("pattern %q vs sample %q: len %d != %d", tc.pattern, sample, len(patSegs), len(samSegs))
			continue
		}
		for i := range patSegs {
			if strings.HasPrefix(patSegs[i], "{") {
				continue
			}
			if patSegs[i] != samSegs[i] {
				t.Errorf("pattern %q seg %d: %q != %q (sample %q)", tc.pattern, i, patSegs[i], samSegs[i], sample)
			}
		}
	}
}

func TestLoginNext(t *testing.T) {
	t.Parallel()
	if got := LoginNext(""); got != Login() {
		t.Fatalf("empty next = %q", got)
	}
	if got := LoginNext(Projects()); !strings.Contains(got, "next=") {
		t.Fatalf("next missing: %q", got)
	}
}

func TestCompileQuery(t *testing.T) {
	t.Parallel()
	got := Compile("abc", "main.tex")
	if !strings.HasPrefix(got, "/api/compile?") {
		t.Fatalf("got %q", got)
	}
	if !strings.Contains(got, "project=abc") || !strings.Contains(got, "file=main.tex") {
		t.Fatalf("query missing: %q", got)
	}
}
