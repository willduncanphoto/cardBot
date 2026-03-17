package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		cmp  int
	}{
		{"0.2.1", "0.2.0", 1},
		{"0.2.0", "0.2.0", 0},
		{"0.1.9", "0.2.0", -1},
		{"v1.0.0", "0.9.9", 1},
		{"1.2", "1.2.0", 0},
	}

	for _, tt := range tests {
		got, err := compareVersions(tt.a, tt.b)
		if err != nil {
			t.Fatalf("compareVersions(%q, %q) error: %v", tt.a, tt.b, err)
		}
		if got != tt.cmp {
			t.Fatalf("compareVersions(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.cmp)
		}
	}
}

func TestParseChecksums(t *testing.T) {
	data := []byte("abcdef1234567890  cardbot-darwin-arm64\n1234  *cardbot-darwin-amd64\n")
	m, err := parseChecksums(data)
	if err != nil {
		t.Fatal(err)
	}
	if m["cardbot-darwin-arm64"] != "abcdef1234567890" {
		t.Fatalf("unexpected hash: %q", m["cardbot-darwin-arm64"])
	}
	if m["cardbot-darwin-amd64"] != "1234" {
		t.Fatalf("unexpected hash: %q", m["cardbot-darwin-amd64"])
	}
}

func TestCheckLatest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v0.2.1",
			"assets":   []any{},
		})
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := CheckLatest(ctx, srv.Client(), srv.URL, "owner/repo", "0.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if !res.Update {
		t.Fatal("expected update to be available")
	}
	if res.Latest != "0.2.1" {
		t.Fatalf("Latest = %q, want 0.2.1", res.Latest)
	}
}

func TestSelfUpdateForPlatform(t *testing.T) {
	newBin := []byte("new-binary")
	h := sha256.Sum256(newBin)
	hash := hex.EncodeToString(h[:])

	var serverURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name": "v0.2.1",
				"assets": []map[string]string{
					{"name": "cardbot-darwin-arm64", "browser_download_url": serverURL + "/assets/cardbot-darwin-arm64"},
					{"name": "checksums.txt", "browser_download_url": serverURL + "/assets/checksums.txt"},
				},
			})
		case "/assets/cardbot-darwin-arm64":
			_, _ = w.Write(newBin)
		case "/assets/checksums.txt":
			_, _ = fmt.Fprintf(w, "%s  cardbot-darwin-arm64\n", hash)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	serverURL = srv.URL

	execPath := filepath.Join(t.TempDir(), "cardbot")
	if err := os.WriteFile(execPath, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	installed, err := SelfUpdateForPlatform(ctx, srv.Client(), srv.URL, "owner/repo", "0.2.0", execPath, "darwin", "arm64")
	if err != nil {
		t.Fatal(err)
	}
	if installed != "0.2.1" {
		t.Fatalf("installed = %q, want 0.2.1", installed)
	}
	got, err := os.ReadFile(execPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(newBin) {
		t.Fatalf("binary mismatch: got %q want %q", string(got), string(newBin))
	}
}

func TestSelfUpdateAlreadyUpToDate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v0.2.0",
			"assets":   []any{},
		})
	}))
	defer srv.Close()

	execPath := filepath.Join(t.TempDir(), "cardbot")
	if err := os.WriteFile(execPath, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := SelfUpdateForPlatform(ctx, srv.Client(), srv.URL, "owner/repo", "0.2.0", execPath, "darwin", "arm64")
	if err == nil || err != ErrAlreadyUpToDate {
		t.Fatalf("expected ErrAlreadyUpToDate, got %v", err)
	}
}
