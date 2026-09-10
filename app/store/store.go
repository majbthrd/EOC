//go:build windows || darwin || linux

// Package store provides an in-memory, non-persistent chat store.
// All data is held in volatile RAM and lost on process exit.
package store

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/majbthrd/EOC/app/types/not"
)

// ────────────────────────────────────────────────────────────────────
//  Types  (identical to the original store.go)
// ────────────────────────────────────────────────────────────────────

type File struct {
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
}

type User struct {
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Plan     string    `json:"plan"`
	CachedAt time.Time `json:"cachedAt"`
}

type Message struct {
	Role              string           `json:"role"`
	Content           string           `json:"content"`
	Thinking          string           `json:"thinking"`
	Stream            bool             `json:"stream"`
	Model             string           `json:"model,omitempty"`
	Attachments       []File           `json:"attachments,omitempty"`
	ToolCalls         []ToolCall       `json:"tool_calls,omitempty"`
	ToolCall          *ToolCall        `json:"tool_call,omitempty"`
	ToolName          string           `json:"tool_name,omitempty"`
	ToolResult        *json.RawMessage `json:"tool_result,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	ThinkingTimeStart *time.Time       `json:"thinkingTimeStart,omitempty" ts_type:"Date | undefined" ts_transform:"__VALUE__ && new Date(__VALUE__)"`
	ThinkingTimeEnd   *time.Time       `json:"thinkingTimeEnd,omitempty" ts_type:"Date | undefined" ts_transform:"__VALUE__ && new Date(__VALUE__)"`
}

type MessageOptions struct {
	Model             string
	Attachments       []File
	Stream            bool
	Thinking          string
	ToolCalls         []ToolCall
	ToolCall          *ToolCall
	ToolResult        *json.RawMessage
	ThinkingTimeStart *time.Time
	ThinkingTimeEnd   *time.Time
}

type ToolCall struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Result    any    `json:"result,omitempty"`
}

type Model struct {
	Model      string     `json:"model"`
	Digest     string     `json:"digest,omitempty"`
	ModifiedAt *time.Time `json:"modified_at,omitempty"`
}

type Chat struct {
	ID           string          `json:"id"`
	Messages     []Message       `json:"messages"`
	Title        string          `json:"title"`
	CreatedAt    time.Time       `json:"created_at"`
	BrowserState json.RawMessage `json:"browser_state,omitempty" ts_type:"BrowserStateData"`
}

type Settings struct {
	Expose            bool
	Browser           bool
	Survey            bool
	Models            string
	Agent             bool
	Tools             bool
	WorkingDir        string
	ContextLength     int
	TurboEnabled      bool
	WebSearchEnabled  bool
	ThinkEnabled      bool
	ThinkLevel        string
	SelectedModel     string
	SidebarOpen       bool
	LastHomeView      string
	AutoUpdateEnabled bool
}

// ────────────────────────────────────────────────────────────────────
//  Constructors  (identical to the original store.go)
// ────────────────────────────────────────────────────────────────────

func NewChat(id string) *Chat {
	return &Chat{
		ID:        id,
		Messages:  []Message{},
		CreatedAt: time.Now(),
	}
}

func NewMessage(role, content string, opts *MessageOptions) Message {
	now := time.Now()
	msg := Message{
		Role:      role,
		Content:   content,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if opts != nil {
		msg.Model = opts.Model
		msg.Attachments = opts.Attachments
		msg.Stream = opts.Stream
		msg.Thinking = opts.Thinking
		msg.ToolCalls = opts.ToolCalls
		msg.ToolCall = opts.ToolCall
		msg.ToolResult = opts.ToolResult
		msg.ThinkingTimeStart = opts.ThinkingTimeStart
		msg.ThinkingTimeEnd = opts.ThinkingTimeEnd
	}

	return msg
}

// ────────────────────────────────────────────────────────────────────
//  Store  (in-memory implementation, same public signatures)
// ────────────────────────────────────────────────────────────────────

type Store struct {
	DBPath string
	mu       sync.RWMutex
	deviceID string
	chat     *Chat
	settings Settings
}

func New() *Store {
	return &Store{
		deviceID: uuid.NewString(),
	}
}

func (s *Store) ID() (string, error) {
	return s.deviceID, nil
}

func (s *Store) HasCompletedFirstRun() (bool, error) {
	return false, nil
}

func (s *Store) SetHasCompletedFirstRun(_ bool) error {
	return nil
}

func (s *Store) Settings() (Settings, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings, nil
}

func (s *Store) SetSettings(settings Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings = settings;
	return nil
}

func (s *Store) Chats() ([]Chat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.chat == nil {
		return nil, nil
	}
	return []Chat{*deepCopyChat(s.chat)}, nil
}

func (s *Store) Chat(id string) (*Chat, error) {
	return s.ChatWithOptions(id, true)
}

func (s *Store) ChatWithOptions(_ string, _ bool) (*Chat, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.chat == nil {
		return nil, not.Found
	}
	return deepCopyChat(s.chat), nil
}

func (s *Store) SetChat(chat Chat) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chat = &chat
	return nil
}

func (s *Store) UpdateLastMessage(_ string, message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.chat == nil || len(s.chat.Messages) == 0 {
		return not.Found
	}
	s.chat.Messages[len(s.chat.Messages)-1] = message
	return nil
}

func (s *Store) AppendMessage(chatID string, message Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.chat == nil {
		s.chat = NewChat(chatID)
	}
	s.chat.Messages = append(s.chat.Messages, message)
	return nil
}

func (s *Store) UpdateChatBrowserState(_ string, state json.RawMessage) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.chat == nil {
		return not.Found
	}
	s.chat.BrowserState = state
	return nil
}

func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.chat = nil
	return nil
}

// ────────────────────────────────────────────────────────────────────
//  Helpers
// ────────────────────────────────────────────────────────────────────

func deepCopyChat(c *Chat) *Chat {
	out := *c
	out.Messages = make([]Message, len(c.Messages))
	for i, m := range c.Messages {
		out.Messages[i] = m
		if len(m.Attachments) > 0 {
			out.Messages[i].Attachments = make([]File, len(m.Attachments))
			copy(out.Messages[i].Attachments, m.Attachments)
		}
		if len(m.ToolCalls) > 0 {
			out.Messages[i].ToolCalls = make([]ToolCall, len(m.ToolCalls))
			copy(out.Messages[i].ToolCalls, m.ToolCalls)
		}
	}
	return &out
}
