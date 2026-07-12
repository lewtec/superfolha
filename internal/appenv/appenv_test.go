package appenv

import "testing"

func TestIsDevelopment(t *testing.T) {
	tests := []struct {
		name  string
		goEnv string
		want  bool
	}{
		{name: "empty", goEnv: "", want: false},
		{name: "development", goEnv: "development", want: true},
		{name: "production", goEnv: "production", want: false},
		{name: "other", goEnv: "staging", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GO_ENV", tt.goEnv)
			if got := IsDevelopment(); got != tt.want {
				t.Fatalf("IsDevelopment() with GO_ENV=%q: got %v, want %v", tt.goEnv, got, tt.want)
			}
		})
	}
}
