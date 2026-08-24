package githubapp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExchange(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/login/oauth/access_token", func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(map[string]string{"access_token": "ghu_test"}); err != nil {
			t.Errorf("encode token: %v", err)
		}
	})
	mux.HandleFunc("/user", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer ghu_test" {
			http.Error(w, "nope", http.StatusUnauthorized)
			return
		}
		if err := json.NewEncoder(w).Encode(Identity{ID: 42, Login: "alice"}); err != nil {
			t.Errorf("encode user: %v", err)
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	cfg := Config{
		ClientID:     "id",
		ClientSecret: "sec",
		APIBase:      srv.URL,
		WebBase:      srv.URL,
		HTTPClient:   srv.Client(),
	}
	id, err := cfg.Exchange("code", "http://localhost/cb")
	if err != nil {
		t.Fatal(err)
	}
	if id.Login != "alice" || id.ID != 42 {
		t.Fatalf("identity = %+v", id)
	}
}
