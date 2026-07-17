package project

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestValidateRepoRelativePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "simple", in: "main.tex", want: "main.tex"},
		{name: "nested", in: "src/chapters/intro.tex", want: filepath.Join("src", "chapters", "intro.tex")},
		{name: "dot slash", in: "./main.tex", want: "main.tex"},
		{name: "clean parent within", in: "src/../main.tex", want: "main.tex"},
		{name: "empty", in: "", wantErr: true},
		{name: "dot only", in: ".", wantErr: true},
		{name: "parent only", in: "..", wantErr: true},
		{name: "escape up", in: "../etc/passwd", wantErr: true},
		{name: "escape nested", in: "foo/../../etc/passwd", wantErr: true},
		{name: "absolute", in: "/etc/passwd", wantErr: true},
		{name: "double slash normalizes", in: "foo//bar.tex", want: filepath.Join("foo", "bar.tex")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ValidateRepoRelativePath(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ValidateRepoRelativePath(%q) = %q, want error", tt.in, got)
				}
				if !errors.Is(err, ErrInvalidPath) {
					t.Fatalf("error = %v, want errors.Is(..., ErrInvalidPath)", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("ValidateRepoRelativePath(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("ValidateRepoRelativePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestSafeRepoPath(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "repos", "proj-1")

	tests := []struct {
		name    string
		user    string
		wantRel string
		wantErr bool
	}{
		{name: "good nested", user: "a/b.tex", wantRel: filepath.Join("a", "b.tex")},
		{name: "traversal", user: "../../etc/passwd", wantErr: true},
		{name: "absolute", user: "/tmp/x", wantErr: true},
		{name: "empty", user: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := safeRepoPath(root, tt.user)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("safeRepoPath(%q) = %q, want error", tt.user, got)
				}
				if !errors.Is(err, ErrInvalidPath) {
					t.Fatalf("error = %v, want errors.Is(..., ErrInvalidPath)", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("safeRepoPath(%q) unexpected error: %v", tt.user, err)
			}
			want := filepath.Join(root, tt.wantRel)
			if got != want {
				t.Fatalf("safeRepoPath(%q) = %q, want %q", tt.user, got, want)
			}
		})
	}
}

func TestDecodeFilePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{name: "plain", in: "main.tex", want: "main.tex"},
		{name: "encoded space", in: "my%20file.tex", want: "my file.tex"},
		{name: "encoded nested", in: "dir%2Fsub%20file.tex", want: filepath.Join("dir", "sub file.tex")},
		{name: "plus is literal", in: "a+b.tex", want: "a+b.tex"},
		{name: "traversal after decode", in: "%2e%2e%2fetc%2fpasswd", wantErr: true},
		{name: "absolute after decode", in: "%2Fetc%2Fpasswd", wantErr: true},
		{name: "empty", in: "", wantErr: true},
		{name: "invalid escape", in: "bad%zz", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := DecodeFilePath(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("DecodeFilePath(%q) = %q, want error", tt.in, got)
				}
				if !errors.Is(err, ErrInvalidPath) {
					t.Fatalf("error = %v, want errors.Is(..., ErrInvalidPath)", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeFilePath(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Fatalf("DecodeFilePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
