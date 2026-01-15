package notifier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewSlackNotifier(t *testing.T) {
	webhookURL := "https://hooks.slack.com/services/test"
	notifier := NewSlackNotifier(webhookURL)

	// NewSlackNotifier always returns non-nil
	if notifier == nil {
		t.Fatal("NewSlackNotifier() returned nil")
		return // unreachable, but helps staticcheck
	}

	if notifier.webhookURL != webhookURL {
		t.Errorf("Expected webhookURL %q, got %q", webhookURL, notifier.webhookURL)
	}

	if notifier.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestSlackNotifier_Send(t *testing.T) {
	// モックサーバーを作成
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		var msg SlackMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if msg.Text != "test message" {
			t.Errorf("Expected text 'test message', got %q", msg.Text)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL)
	err := notifier.Send("test message")
	if err != nil {
		t.Errorf("Send() error = %v, want nil", err)
	}
}

func TestSlackNotifier_SendMessage_WithAttachments(t *testing.T) {
	// モックサーバーを作成
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg SlackMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			t.Errorf("Failed to decode request body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if len(msg.Attachments) != 1 {
			t.Errorf("Expected 1 attachment, got %d", len(msg.Attachments))
		}

		attachment := msg.Attachments[0]
		if attachment.Color != "good" {
			t.Errorf("Expected color 'good', got %q", attachment.Color)
		}

		if attachment.Title != "Test Title" {
			t.Errorf("Expected title 'Test Title', got %q", attachment.Title)
		}

		if len(attachment.Fields) != 2 {
			t.Errorf("Expected 2 fields, got %d", len(attachment.Fields))
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL)
	msg := &SlackMessage{
		Attachments: []SlackAttachment{
			{
				Color: "good",
				Title: "Test Title",
				Fields: []SlackAttachmentField{
					{Title: "Field1", Value: "Value1", Short: true},
					{Title: "Field2", Value: "Value2", Short: false},
				},
				Timestamp: time.Now().Unix(),
			},
		},
	}

	err := notifier.SendMessage(msg)
	if err != nil {
		t.Errorf("SendMessage() error = %v, want nil", err)
	}
}

func TestSlackNotifier_Send_ServerError(t *testing.T) {
	// エラーを返すモックサーバー
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	notifier := NewSlackNotifier(server.URL)
	err := notifier.Send("test message")
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

func TestSlackNotifier_Send_InvalidURL(t *testing.T) {
	notifier := NewSlackNotifier("http://invalid-url-that-does-not-exist-12345.com")
	// タイムアウトを短く設定
	notifier.httpClient.Timeout = 100 * time.Millisecond

	err := notifier.Send("test message")
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestSlackMessage_JSON(t *testing.T) {
	msg := SlackMessage{
		Text: "test",
		Attachments: []SlackAttachment{
			{
				Color: "good",
				Title: "Test",
				Fields: []SlackAttachmentField{
					{Title: "Field", Value: "Value", Short: true},
				},
				Timestamp: 1234567890,
			},
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal SlackMessage: %v", err)
	}

	var decoded SlackMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal SlackMessage: %v", err)
	}

	if decoded.Text != msg.Text {
		t.Errorf("Expected text %q, got %q", msg.Text, decoded.Text)
	}

	if len(decoded.Attachments) != len(msg.Attachments) {
		t.Errorf("Expected %d attachments, got %d", len(msg.Attachments), len(decoded.Attachments))
	}

	if decoded.Attachments[0].Color != msg.Attachments[0].Color {
		t.Errorf("Expected color %q, got %q", msg.Attachments[0].Color, decoded.Attachments[0].Color)
	}
}
