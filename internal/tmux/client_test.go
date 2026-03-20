package tmux

import "testing"

func TestSessionName(t *testing.T) {
	tests := []struct {
		group    string
		expected string
	}{
		{"backend", "muxrun-backend"},
		{"frontend", "muxrun-frontend"},
		{"my-app", "muxrun-my-app"},
	}
	for _, tt := range tests {
		got := SessionName(tt.group)
		if got != tt.expected {
			t.Errorf("SessionName(%q) = %q, want %q", tt.group, got, tt.expected)
		}
	}
}

func TestGroupName(t *testing.T) {
	tests := []struct {
		session  string
		expected string
	}{
		{"muxrun-backend", "backend"},
		{"muxrun-frontend", "frontend"},
	}
	for _, tt := range tests {
		got := GroupName(tt.session)
		if got != tt.expected {
			t.Errorf("GroupName(%q) = %q, want %q", tt.session, got, tt.expected)
		}
	}
}

func TestIsMuxrunSession(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"muxrun-backend", true},
		{"muxrun-x", true},
		{"muxrun-", false},
		{"other-session", false},
		{"", false},
	}
	for _, tt := range tests {
		got := IsMuxrunSession(tt.name)
		if got != tt.expected {
			t.Errorf("IsMuxrunSession(%q) = %v, want %v", tt.name, got, tt.expected)
		}
	}
}
