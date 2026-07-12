package hermes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/naseer2426/split-bot-whatsapp/internal/config"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("DATABASE_URL", "postgres://localhost/test")
	_ = os.Setenv("SPLIT_BOT_URL", "http://localhost:9001")
	_ = os.Setenv("NANOBOT_URL", "http://localhost:9002")
	_ = os.Setenv("HERMES_URL", "http://localhost:8765")
	_ = os.Setenv("BOT_NAME", "testbot")
	_ = os.Setenv("PORT", "8080")
	config.MustLoad()
	os.Exit(m.Run())
}

func TestSendMessageJSONAndBearer(t *testing.T) {
	var gotAuth string
	var gotBody MessageRequest

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/message" {
			t.Errorf("path = %s, want /message", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	setTestHermesConfig(srv.URL, "secret-key")
	t.Cleanup(func() { setTestHermesConfig("http://localhost:8765", "") })

	err := SendMessage(MessageRequest{
		Sender: "6012@s.whatsapp.net",
		ChatID: "120363@g.us",
		Text:   "hello",
		Media: []MediaItem{{
			Type: "image",
			Data: "abc123",
			Mime: "image/jpeg",
		}},
		MessageID: "msgid",
		IsMention: true,
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}

	if gotAuth != "Bearer secret-key" {
		t.Fatalf("Authorization = %q, want %q", gotAuth, "Bearer secret-key")
	}
	if gotBody.Text != "hello" || gotBody.ChatID != "120363@g.us" {
		t.Fatalf("unexpected body: %+v", gotBody)
	}
	if len(gotBody.Media) != 1 || gotBody.Media[0].Type != "image" || gotBody.Media[0].Data != "abc123" {
		t.Fatalf("unexpected media: %+v", gotBody.Media)
	}
	if !gotBody.IsMention || gotBody.MessageID != "msgid" {
		t.Fatalf("unexpected mention/id: %+v", gotBody)
	}
}

func TestSendMessageWithoutAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	setTestHermesConfig(srv.URL, "")
	t.Cleanup(func() { setTestHermesConfig("http://localhost:8765", "") })

	if err := SendMessage(MessageRequest{
		Sender: "a@s.whatsapp.net",
		ChatID: "a@s.whatsapp.net",
		Text:   "hi",
	}); err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if gotAuth != "" {
		t.Fatalf("Authorization = %q, want empty", gotAuth)
	}
}

func TestSendMessageNonOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"ok":false,"error":"bad"}`))
	}))
	defer srv.Close()

	setTestHermesConfig(srv.URL, "")
	t.Cleanup(func() { setTestHermesConfig("http://localhost:8765", "") })

	err := SendMessage(MessageRequest{Sender: "a", ChatID: "a", Text: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "non-success status") {
		t.Fatalf("error = %v, want non-success status", err)
	}
}

func setTestHermesConfig(url, apiKey string) {
	cfg := config.Get()
	cfg.Hermes.URL = url
	cfg.Hermes.APIKey = apiKey
}
