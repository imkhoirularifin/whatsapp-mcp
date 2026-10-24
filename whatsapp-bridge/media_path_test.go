package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckMediaPathRejectsTraversal(t *testing.T) {
	tmp := t.TempDir()
	store := filepath.Join(tmp, "store")
	if err := os.MkdirAll(store, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(tmp, "ok.pdf")
	if err := os.WriteFile(outside, []byte("pdf"), 0o644); err != nil {
		t.Fatal(err)
	}
	secret := filepath.Join(store, "whatsapp.db")
	if err := os.WriteFile(secret, []byte("db"), 0o644); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name    string
		path    string
		roots   []string
		wantErr string
	}{
		{
			// filepath.Join cleans ".."; keep it literal so the CWE-22 check fires
			name:    "dot-dot refused",
			path:    filepath.Join(tmp, "safe") + "/../store/whatsapp.db",
			wantErr: `".."`,
		},
		{
			name:    "store refused",
			path:    secret,
			wantErr: "store/",
		},
		{
			name:    "hidden ssh-style path refused",
			path:    filepath.Join(tmp, ".ssh", "id_ed25519"),
			wantErr: "hidden path component",
		},
		{
			name:  "ordinary file allowed without roots",
			path:  outside,
			roots: nil,
		},
		{
			name:    "outside media roots refused",
			path:    outside,
			roots:   []string{filepath.Join(tmp, "allowed")},
			wantErr: "WHATSAPP_MEDIA_ROOTS",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if strings.Contains(tc.wantErr, "hidden") {
				if err := os.MkdirAll(filepath.Dir(tc.path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(tc.path, []byte("key"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			err := checkMediaPath(tc.path, tc.roots, store)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("error = %v, want substring %q", err, tc.wantErr)
			}
		})
	}
}

func TestCheckMediaPathAllowsUnderMediaRoots(t *testing.T) {
	tmp := t.TempDir()
	store := filepath.Join(tmp, "store")
	root := filepath.Join(tmp, "allowed")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "doc.pdf")
	if err := os.WriteFile(file, []byte("pdf"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkMediaPath(file, []string{root}, store); err != nil {
		t.Fatal(err)
	}
}
