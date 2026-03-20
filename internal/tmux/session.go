package tmux

import "fmt"

const sessionPrefix = "muxrun-"

func SessionName(group string) string {
	return sessionPrefix + group
}

func GroupName(session string) string {
	if len(session) > len(sessionPrefix) {
		return session[len(sessionPrefix):]
	}
	return session
}

func IsMuxrunSession(name string) bool {
	return len(name) > len(sessionPrefix) && name[:len(sessionPrefix)] == sessionPrefix
}

// EnsureSession creates a session if it doesn't exist.
func EnsureSession(c Client, name string) error {
	exists, err := c.HasSession(name)
	if err != nil {
		return fmt.Errorf("checking session %q: %w", name, err)
	}
	if exists {
		return nil
	}
	return c.NewSession(name)
}
