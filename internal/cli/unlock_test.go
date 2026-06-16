package cli

import (
	"encoding/base64"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSessionActivityTracking(t *testing.T) {
	dir := t.TempDir()
	oldTmp := os.Getenv("TMPDIR")
	os.Setenv("TMPDIR", dir)
	defer os.Setenv("TMPDIR", oldTmp)

	key := []byte("test-session-key-32-bytes-long!!")
	if err := writeSessionKey(key); err != nil {
		t.Fatal(err)
	}

	idle, err := sessionIdleSince()
	if err != nil {
		t.Fatal(err)
	}
	if idle > 2*time.Second {
		t.Fatalf("expected fresh session, idle = %v", idle)
	}

	// Simulate 20 minutes without use.
	path := sessionKeyPath()
	data, _ := os.ReadFile(path)
	lines := splitLines(string(data))
	past := time.Now().Add(-20 * time.Minute).Unix()
	if len(lines) >= 3 {
		lines[2] = strconv.FormatInt(past, 10)
	}
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)

	idle, err = sessionIdleSince()
	if err != nil {
		t.Fatal(err)
	}
	if idle < 19*time.Minute {
		t.Fatalf("expected ~20m idle, got %v", idle)
	}

	if err := touchSessionActivity(); err != nil {
		t.Fatal(err)
	}
	idle, err = sessionIdleSince()
	if err != nil {
		t.Fatal(err)
	}
	if idle > 2*time.Second {
		t.Fatalf("touchSessionActivity should reset idle, got %v", idle)
	}
}

func TestRefreshSessionKeyDoesNotResetActivity(t *testing.T) {
	dir := t.TempDir()
	oldTmp := os.Getenv("TMPDIR")
	os.Setenv("TMPDIR", dir)
	defer os.Setenv("TMPDIR", oldTmp)

	key := []byte("test-session-key-32-bytes-long!!")
	if err := writeSessionKey(key); err != nil {
		t.Fatal(err)
	}

	path := sessionKeyPath()
	data, _ := os.ReadFile(path)
	lines := splitLines(string(data))
	past := time.Now().Add(-10 * time.Minute).Unix()
	lines[2] = strconv.FormatInt(past, 10)
	os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600)

	if err := refreshSessionKey(key); err != nil {
		t.Fatal(err)
	}

	idle, err := sessionIdleSince()
	if err != nil {
		t.Fatal(err)
	}
	if idle < 9*time.Minute {
		t.Fatalf("refresh should not bump activity, idle = %v", idle)
	}

	data, _ = os.ReadFile(path)
	lines = splitLines(string(data))
	expiry, _ := strconv.ParseInt(lines[0], 10, 64)
	if expiry <= time.Now().Unix() {
		t.Fatal("expected extended expiry")
	}
}

func TestReadSessionKeyLegacyTwoLineFormat(t *testing.T) {
	dir := t.TempDir()
	oldTmp := os.Getenv("TMPDIR")
	os.Setenv("TMPDIR", dir)
	defer os.Setenv("TMPDIR", oldTmp)

	path := sessionKeyPath()
	key := []byte("legacy-key")
	content := strconv.FormatInt(time.Now().Add(time.Hour).Unix(), 10) + "\n" +
		base64.StdEncoding.EncodeToString(key)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := readSessionKey()
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(key) {
		t.Fatalf("key = %q", got)
	}
}
