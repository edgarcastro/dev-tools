package envguard

import "testing"

func TestBlocked(t *testing.T) {
	cases := []struct {
		in      string
		blocked bool
	}{
		{".env", true},
		{".env.local", true},
		{"src/.env", true},
		{"/app/.env.production", true},
		{"cat .env*", true},
		{"--env-file=.env", true},
		{"source ./.env", true},
		{"echo hi && cat .env", true},
		{".env.example.local", true},
		{".env.example*", true},
		{".env.example", false},
		{"./.env.example", false},
		{"cat .env.example", false},
		{"cp .env.example .env.example", false},
		{"cp .env.example .env", true},
		{"foo.environment", false},
		{"src/.environment", false},
		{"environment", false},
		{"", false},
	}
	for _, c := range cases {
		if got := Blocked([]string{c.in}); got != c.blocked {
			t.Errorf("Blocked(%q) = %v, want %v", c.in, got, c.blocked)
		}
	}
}

func TestTargets(t *testing.T) {
	got, err := Targets([]byte(`{"tool_name":"Read","tool_input":{"file_path":"a/.env","limit":3}}`))
	if err != nil || len(got) != 1 || got[0] != "a/.env" {
		t.Fatalf("Targets = %v, %v", got, err)
	}
	if !Blocked(got) {
		t.Fatal("expected blocked")
	}
}
