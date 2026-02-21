package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
)

// APICall records a single Slack API call made to the mock server.
type APICall struct {
	Method string
	Params map[string]string
}

// MockSlackServer wraps an httptest.Server that responds to Slack API methods
// with canned responses and records every call for later assertion.
type MockSlackServer struct {
	Server *httptest.Server

	mu      sync.Mutex
	calls   []APICall
	options MockSlackOptions
}

// MockSlackOptions configures canned responses for the mock server.
type MockSlackOptions struct {
	// conversations.info
	ChannelName string // returned as NameNormalized

	// conversations.history — keyed by ChannelID
	HistoryMessages map[string][]HistoryMessage

	// users.info
	UserTZ string

	// search.messages
	SearchTotalCount int

	// Custom handler overrides (optional)
	PostMessageHook func(params map[string]string)
}

// HistoryMessage is a simplified message for canned history responses.
type HistoryMessage struct {
	User      string
	Text      string
	Timestamp string
	Channel   string
	Reactions []HistoryReaction
}

// HistoryReaction is a simplified reaction for canned history responses.
type HistoryReaction struct {
	Name  string
	Users []string
}

// NewMockSlackServer creates and starts a mock Slack API server.
func NewMockSlackServer(opts MockSlackOptions) *MockSlackServer {
	m := &MockSlackServer{options: opts}

	mux := http.NewServeMux()
	mux.HandleFunc("/conversations.info", m.handleConversationsInfo)
	mux.HandleFunc("/conversations.join", m.handleConversationsJoin)
	mux.HandleFunc("/conversations.history", m.handleConversationsHistory)
	mux.HandleFunc("/chat.postMessage", m.handleChatPostMessage)
	mux.HandleFunc("/chat.delete", m.handleChatDelete)
	mux.HandleFunc("/reactions.add", m.handleReactionsAdd)
	mux.HandleFunc("/users.info", m.handleUsersInfo)
	mux.HandleFunc("/search.messages", m.handleSearchMessages)

	m.Server = httptest.NewServer(mux)
	return m
}

// Close shuts down the mock server.
func (m *MockSlackServer) Close() {
	m.Server.Close()
}

// URL returns the base URL with a trailing slash, suitable for slack.OptionAPIURL().
func (m *MockSlackServer) URL() string {
	return m.Server.URL + "/"
}

// Calls returns a copy of all recorded API calls.
func (m *MockSlackServer) Calls() []APICall {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]APICall, len(m.calls))
	copy(out, m.calls)
	return out
}

// CallsFor returns all recorded calls matching the given method name.
func (m *MockSlackServer) CallsFor(method string) []APICall {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []APICall
	for _, c := range m.calls {
		if c.Method == method {
			out = append(out, c)
		}
	}
	return out
}

// Reset clears all recorded calls.
func (m *MockSlackServer) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = nil
}

func (m *MockSlackServer) record(method string, params map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = append(m.calls, APICall{Method: method, Params: params})
}

func (m *MockSlackServer) parseForm(r *http.Request) map[string]string {
	_ = r.ParseForm()
	params := make(map[string]string)
	for k, v := range r.Form {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	return params
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (m *MockSlackServer) handleConversationsInfo(w http.ResponseWriter, r *http.Request) {
	params := m.parseForm(r)
	m.record("conversations.info", params)

	writeJSON(w, map[string]interface{}{
		"ok": true,
		"channel": map[string]interface{}{
			"id":              params["channel"],
			"name":            m.options.ChannelName,
			"name_normalized": m.options.ChannelName,
		},
	})
}

func (m *MockSlackServer) handleConversationsJoin(w http.ResponseWriter, r *http.Request) {
	params := m.parseForm(r)
	m.record("conversations.join", params)

	writeJSON(w, map[string]interface{}{
		"ok": true,
		"channel": map[string]interface{}{
			"id": params["channel"],
		},
	})
}

func (m *MockSlackServer) handleConversationsHistory(w http.ResponseWriter, r *http.Request) {
	params := m.parseForm(r)
	m.record("conversations.history", params)

	channelID := params["channel"]
	msgs, ok := m.options.HistoryMessages[channelID]
	if !ok {
		writeJSON(w, map[string]interface{}{
			"ok":       true,
			"messages": []interface{}{},
		})
		return
	}

	var jsonMsgs []map[string]interface{}
	for _, msg := range msgs {
		jm := map[string]interface{}{
			"user": msg.User,
			"text": msg.Text,
			"ts":   msg.Timestamp,
		}
		if len(msg.Reactions) > 0 {
			var reactions []map[string]interface{}
			for _, rx := range msg.Reactions {
				reactions = append(reactions, map[string]interface{}{
					"name":  rx.Name,
					"users": rx.Users,
				})
			}
			jm["reactions"] = reactions
		}
		jsonMsgs = append(jsonMsgs, jm)
	}

	writeJSON(w, map[string]interface{}{
		"ok":       true,
		"messages": jsonMsgs,
	})
}

func (m *MockSlackServer) handleChatPostMessage(w http.ResponseWriter, r *http.Request) {
	params := m.parseForm(r)
	m.record("chat.postMessage", params)

	if m.options.PostMessageHook != nil {
		m.options.PostMessageHook(params)
	}

	writeJSON(w, map[string]interface{}{
		"ok":      true,
		"channel": params["channel"],
		"ts":      "1234567890.000100",
	})
}

func (m *MockSlackServer) handleChatDelete(w http.ResponseWriter, r *http.Request) {
	params := m.parseForm(r)
	m.record("chat.delete", params)

	writeJSON(w, map[string]interface{}{
		"ok":      true,
		"channel": params["channel"],
		"ts":      params["ts"],
	})
}

func (m *MockSlackServer) handleReactionsAdd(w http.ResponseWriter, r *http.Request) {
	params := m.parseForm(r)
	m.record("reactions.add", params)

	writeJSON(w, map[string]interface{}{
		"ok": true,
	})
}

func (m *MockSlackServer) handleUsersInfo(w http.ResponseWriter, r *http.Request) {
	params := m.parseForm(r)
	m.record("users.info", params)

	writeJSON(w, map[string]interface{}{
		"ok": true,
		"user": map[string]interface{}{
			"id": params["user"],
			"tz": m.options.UserTZ,
		},
	})
}

func (m *MockSlackServer) handleSearchMessages(w http.ResponseWriter, r *http.Request) {
	params := m.parseForm(r)
	m.record("search.messages", params)

	writeJSON(w, map[string]interface{}{
		"ok": true,
		"messages": map[string]interface{}{
			"pagination": map[string]interface{}{
				"total_count": m.options.SearchTotalCount,
			},
			"matches": []interface{}{},
		},
	})
}
