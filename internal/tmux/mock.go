package tmux

// MockClient is a test double for tmux.Client.
type MockClient struct {
	Sessions map[string][]Window
}

func NewMockClient() *MockClient {
	return &MockClient{Sessions: make(map[string][]Window)}
}

func (m *MockClient) HasSession(name string) (bool, error) {
	_, ok := m.Sessions[name]
	return ok, nil
}

func (m *MockClient) NewSession(name string) error {
	if _, ok := m.Sessions[name]; !ok {
		m.Sessions[name] = []Window{{Name: InitWindowName}}
	}
	return nil
}

func (m *MockClient) KillSession(name string) error {
	delete(m.Sessions, name)
	return nil
}

func (m *MockClient) ListSessions() ([]Session, error) {
	var sessions []Session
	for name := range m.Sessions {
		sessions = append(sessions, Session{Name: name})
	}
	return sessions, nil
}

func (m *MockClient) NewWindow(session, window, dir string) error {
	m.Sessions[session] = append(m.Sessions[session], Window{Name: window})
	return nil
}

func (m *MockClient) KillWindow(session, window string) error {
	windows := m.Sessions[session]
	for i, w := range windows {
		if w.Name == window {
			m.Sessions[session] = append(windows[:i], windows[i+1:]...)
			return nil
		}
	}
	return nil
}

func (m *MockClient) ListWindows(session string) ([]Window, error) {
	return m.Sessions[session], nil
}

func (m *MockClient) SendKeys(session, window, keys string) error {
	return nil
}

func (m *MockClient) GetPanePID(session, window string) (int, error) {
	for _, w := range m.Sessions[session] {
		if w.Name == window {
			return w.PID, nil
		}
	}
	return 0, nil
}
