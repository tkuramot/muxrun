package tmux

import (
	"testing"
	"time"
)

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

func TestParseWindowLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		ok       bool
		expected Window
	}{
		{
			name:     "running window",
			line:     "api\t123\t0\t0\t\t0\t/tmp/app",
			ok:       true,
			expected: Window{Name: "api", PID: 123, Dir: "/tmp/app"},
		},
		{
			name:     "dead window",
			line:     "api\t123\t1\t7\t\t1700000000\t/tmp/app",
			ok:       true,
			expected: Window{Name: "api", PID: 123, Dead: true, DeadStatus: 7, DeadTime: time.Unix(1700000000, 0), Dir: "/tmp/app"},
		},
		{
			name:     "signalled window has no exit status",
			line:     "api\t123\t1\t\tint\t1700000000\t/tmp/app",
			ok:       true,
			expected: Window{Name: "api", PID: 123, Dead: true, DeadSignal: "int", DeadTime: time.Unix(1700000000, 0), Dir: "/tmp/app"},
		},
		{
			name:     "path containing spaces",
			line:     "api\t123\t1\t7\t\t1700000000\t/tmp/my project dir",
			ok:       true,
			expected: Window{Name: "api", PID: 123, Dead: true, DeadStatus: 7, DeadTime: time.Unix(1700000000, 0), Dir: "/tmp/my project dir"},
		},
		{
			name:     "missing trailing fields",
			line:     "api\t123",
			ok:       true,
			expected: Window{Name: "api", PID: 123},
		},
		{
			name: "not enough fields",
			line: "api",
			ok:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseWindowLine(tt.line)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if got != tt.expected {
				t.Errorf("parseWindowLine(%q) = %+v, want %+v", tt.line, got, tt.expected)
			}
		})
	}
}

func TestShellCommand(t *testing.T) {
	tests := []struct {
		name     string
		shell    string
		cmd      string
		expected string
	}{
		{
			name:     "login shell gets rc files",
			shell:    "/bin/zsh",
			cmd:      "npm run dev",
			expected: `exec '/bin/zsh' -lic 'npm run dev'`,
		},
		{
			name:     "unknown shell runs with -c only",
			shell:    "/bin/sh",
			cmd:      "npm run dev",
			expected: `exec '/bin/sh' -c 'npm run dev'`,
		},
		{
			name:     "single quotes in the command are escaped",
			shell:    "/bin/bash",
			cmd:      `echo 'hi there'`,
			expected: `exec '/bin/bash' -lic 'echo '\''hi there'\'''`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &client{shell: tt.shell}
			if got := c.shellCommand(tt.cmd); got != tt.expected {
				t.Errorf("shellCommand(%q) = %q, want %q", tt.cmd, got, tt.expected)
			}
		})
	}
}
