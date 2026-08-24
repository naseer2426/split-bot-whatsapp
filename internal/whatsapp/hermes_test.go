package whatsapp

import (
	"testing"

	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

func TestHermesMessageTextUsesImageCaption(t *testing.T) {
	evt := &events.Message{
		Message: &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption: proto.String("look at this"),
			},
		},
	}
	if got := hermesMessageText(evt); got != "look at this" {
		t.Fatalf("hermesMessageText = %q, want image caption", got)
	}
}

func TestHermesMessageTextUsesDocumentCaption(t *testing.T) {
	evt := &events.Message{
		Message: &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				Caption:  proto.String("please review"),
				FileName: proto.String("report.pdf"),
			},
		},
	}
	if got := hermesMessageText(evt); got != "please review" {
		t.Fatalf("hermesMessageText = %q, want document caption", got)
	}
}

func TestHermesMessageTextPrefersConversationOverCaption(t *testing.T) {
	evt := &events.Message{
		Message: &waProto.Message{
			Conversation: proto.String("hello"),
			ImageMessage: &waProto.ImageMessage{
				Caption: proto.String("caption"),
			},
		},
	}
	if got := hermesMessageText(evt); got != "hello" {
		t.Fatalf("hermesMessageText = %q, want conversation text", got)
	}
}

func TestHasHermesInboundMedia(t *testing.T) {
	if hasHermesInboundMedia(&events.Message{Message: &waProto.Message{
		Conversation: proto.String("hi"),
	}}) {
		t.Fatal("text-only message should not have media")
	}
	if !hasHermesInboundMedia(&events.Message{Message: &waProto.Message{
		ImageMessage: &waProto.ImageMessage{},
	}}) {
		t.Fatal("image message should have media")
	}
	if !hasHermesInboundMedia(&events.Message{Message: &waProto.Message{
		DocumentMessage: &waProto.DocumentMessage{FileName: proto.String("a.pdf")},
	}}) {
		t.Fatal("document message should have media")
	}
}

func TestQuotedMessageIDFromDocument(t *testing.T) {
	evt := &events.Message{
		Message: &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				Caption: proto.String("see attached"),
				ContextInfo: &waProto.ContextInfo{
					StanzaID: proto.String("quoted-doc"),
				},
			},
		},
	}
	if got := quotedMessageID(evt); got != "quoted-doc" {
		t.Fatalf("quotedMessageID = %q, want quoted-doc", got)
	}
}

func TestHermesMediaKind(t *testing.T) {
	tests := []struct {
		mime     string
		fallback string
		want     string
	}{
		{"image/jpeg", "document", "image"},
		{"image/png; charset=binary", "document", "image"},
		{"application/pdf", "document", "document"},
		{"", "document", "document"},
		{"video/mp4", "document", "video"},
		{"audio/ogg", "document", "audio"},
	}
	for _, tt := range tests {
		if got := hermesMediaKind(tt.mime, tt.fallback); got != tt.want {
			t.Errorf("hermesMediaKind(%q, %q) = %q, want %q", tt.mime, tt.fallback, got, tt.want)
		}
	}
}

func TestShouldProcessHermesMsgGroupMediaWithoutMention(t *testing.T) {
	h := &Handler{botName: "testbot"}
	evt := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{
				Chat: types.NewJID("120363", types.GroupServer),
			},
		},
	}
	if !h.shouldProcessHermesMsg(evt, "please review", true) {
		t.Fatal("group document/image should be processed without a bot mention")
	}
	if h.shouldProcessHermesMsg(evt, "hello everyone", false) {
		t.Fatal("group text without mention or media should be skipped")
	}
	if !h.shouldProcessHermesMsg(evt, "@testbot hello", false) {
		t.Fatal("group mention without media should be processed")
	}
}
