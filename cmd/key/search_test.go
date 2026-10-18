package main

import (
	"testing"

	"gitee.com/kwelldo/secrets"
)

func TestSearchEntries(t *testing.T) {
	deep := &secrets.Entry{
		Name:  "deepseek_api_key",
		Value: "sk-secret-value",
		Notes: []secrets.Note{{Label: "base_url", Value: "https://api.deepseek.com"}},
	}
	gh := &secrets.Entry{
		Name:  "github_token",
		Value: "ghp-secret",
		Notes: []secrets.Note{{Label: "note", Value: "read-only"}},
	}
	entries := []*secrets.Entry{deep, gh}

	// Case-insensitive name match, plus the note value that contains the
	// query: one entry can yield one row per match.
	ms := searchEntries(entries, "DEEPSEEK")
	if len(ms) != 2 {
		t.Fatalf("search(DEEPSEEK) = %d rows, want 2 (name + note value)", len(ms))
	}
	if ms[0].entry.Name != "deepseek_api_key" || ms[0].field != "" {
		t.Errorf("name match row = %s %q", ms[0].entry.Name, ms[0].field)
	}
	if ms[1].field != "base_url=https://api.deepseek.com" {
		t.Errorf("note match row = %q", ms[1].field)
	}

	// Note label match.
	ms = searchEntries(entries, "base_url")
	if len(ms) != 1 || ms[0].field != "base_url=https://api.deepseek.com" {
		t.Errorf("search(base_url) = %+v", ms)
	}

	// Values are never searched.
	if ms := searchEntries(entries, "sk-secret"); len(ms) != 0 {
		t.Errorf("search(sk-secret) must not match the value, got %+v", ms)
	}
	if ms := searchEntries(entries, "ghp-"); len(ms) != 0 {
		t.Errorf("search(ghp-) must not match the value, got %+v", ms)
	}

	// No match.
	if ms := searchEntries(entries, "zzz"); len(ms) != 0 {
		t.Errorf("search(zzz) = %+v, want none", ms)
	}
}
