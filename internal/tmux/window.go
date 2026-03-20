package tmux

// HasWindow checks if a window exists in the session.
func HasWindow(c Client, session, window string) (bool, error) {
	windows, err := c.ListWindows(session)
	if err != nil {
		return false, err
	}
	for _, w := range windows {
		if w.Name == window {
			return true, nil
		}
	}
	return false, nil
}
