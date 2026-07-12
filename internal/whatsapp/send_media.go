package whatsapp

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

const maxMediaBytes = 16 << 20 // 16 MiB

// SendMediaToChatParams is the input for sending media to a WhatsApp chat.
type SendMediaToChatParams struct {
	ChatID     string
	MediaType  string // image|video|audio|document
	DataBase64 string
	FileURL    string
	Mime       string
	Caption    string
	Filename   string
	ReplyTo    string
}

// SendMediaToChat uploads and sends media to a WhatsApp chat identified by a full JID.
func (h *Handler) SendMediaToChat(ctx context.Context, p SendMediaToChatParams) error {
	jid, err := types.ParseJID(p.ChatID)
	if err != nil {
		return fmt.Errorf("invalid chat JID %q: %w", p.ChatID, err)
	}

	data, err := resolveMediaBytes(ctx, p.DataBase64, p.FileURL)
	if err != nil {
		return err
	}

	mediaType, err := mapWhatsmeowMediaType(p.MediaType)
	if err != nil {
		return err
	}

	mime := strings.TrimSpace(p.Mime)
	if mime == "" {
		mime = defaultMimeForMedia(p.MediaType)
	}

	uploaded, err := h.client.Upload(ctx, data, mediaType)
	if err != nil {
		return fmt.Errorf("failed to upload media: %w", err)
	}

	msg, err := buildMediaMessage(p.MediaType, mime, p.Caption, p.Filename, p.ReplyTo, jid, uploaded)
	if err != nil {
		return err
	}

	_, err = h.client.SendMessage(ctx, jid, msg)
	if err != nil {
		return fmt.Errorf("failed to send media to chat %s: %w", p.ChatID, err)
	}

	_ = h.stopTyping(ctx, jid)
	fmt.Printf("Sent %s media to chat %s (%d bytes)\n", p.MediaType, p.ChatID, len(data))
	return nil
}

// SendTyping sets composing presence for a chat identified by a full JID.
func (h *Handler) SendTyping(ctx context.Context, chatID string) error {
	jid, err := types.ParseJID(chatID)
	if err != nil {
		return fmt.Errorf("invalid chat JID %q: %w", chatID, err)
	}
	return h.typing(ctx, jid)
}

func resolveMediaBytes(ctx context.Context, dataBase64, fileURL string) ([]byte, error) {
	dataBase64 = strings.TrimSpace(dataBase64)
	fileURL = strings.TrimSpace(fileURL)

	switch {
	case dataBase64 != "":
		data, err := base64.StdEncoding.DecodeString(dataBase64)
		if err != nil {
			// Some clients omit padding; try raw encoding as a fallback.
			data, err = base64.RawStdEncoding.DecodeString(dataBase64)
			if err != nil {
				return nil, fmt.Errorf("invalid data_base64: %w", err)
			}
		}
		if len(data) == 0 {
			return nil, fmt.Errorf("data_base64 decoded to empty payload")
		}
		if len(data) > maxMediaBytes {
			return nil, fmt.Errorf("media exceeds max size of %d bytes", maxMediaBytes)
		}
		return data, nil
	case fileURL != "":
		return downloadMediaURL(ctx, fileURL)
	default:
		return nil, fmt.Errorf("either data_base64 or file_url is required")
	}
}

func downloadMediaURL(ctx context.Context, fileURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid file_url: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file_url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("file_url returned HTTP %d", resp.StatusCode)
	}

	limited := io.LimitReader(resp.Body, maxMediaBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("failed to read file_url body: %w", err)
	}
	if len(data) > maxMediaBytes {
		return nil, fmt.Errorf("media exceeds max size of %d bytes", maxMediaBytes)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("file_url returned empty body")
	}
	return data, nil
}

func mapWhatsmeowMediaType(mediaType string) (whatsmeow.MediaType, error) {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image":
		return whatsmeow.MediaImage, nil
	case "video":
		return whatsmeow.MediaVideo, nil
	case "audio":
		return whatsmeow.MediaAudio, nil
	case "document":
		return whatsmeow.MediaDocument, nil
	default:
		return "", fmt.Errorf("unsupported media_type %q (valid: image, video, audio, document)", mediaType)
	}
}

func defaultMimeForMedia(mediaType string) string {
	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image":
		return "image/jpeg"
	case "video":
		return "video/mp4"
	case "audio":
		return "audio/ogg; codecs=opus"
	default:
		return "application/octet-stream"
	}
}

func buildMediaMessage(
	mediaType, mime, caption, filename, replyTo string,
	chat types.JID,
	uploaded whatsmeow.UploadResponse,
) (*waProto.Message, error) {
	var ctxInfo *waProto.ContextInfo
	if replyTo != "" {
		ctxInfo = &waProto.ContextInfo{
			StanzaID:  proto.String(replyTo),
			RemoteJID: proto.String(chat.String()),
		}
	}

	switch strings.ToLower(strings.TrimSpace(mediaType)) {
	case "image":
		img := &waProto.ImageMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			Mimetype:      proto.String(mime),
			ContextInfo:   ctxInfo,
		}
		if caption != "" {
			img.Caption = proto.String(caption)
		}
		return &waProto.Message{ImageMessage: img}, nil

	case "video":
		vid := &waProto.VideoMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			Mimetype:      proto.String(mime),
			ContextInfo:   ctxInfo,
		}
		if caption != "" {
			vid.Caption = proto.String(caption)
		}
		return &waProto.Message{VideoMessage: vid}, nil

	case "audio":
		return &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(uploaded.FileLength),
				Mimetype:      proto.String(mime),
				ContextInfo:   ctxInfo,
			},
		}, nil

	case "document":
		if filename == "" {
			filename = "file"
		}
		doc := &waProto.DocumentMessage{
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
			Mimetype:      proto.String(mime),
			FileName:      proto.String(filename),
			ContextInfo:   ctxInfo,
		}
		if caption != "" {
			doc.Caption = proto.String(caption)
		}
		return &waProto.Message{DocumentMessage: doc}, nil

	default:
		return nil, fmt.Errorf("unsupported media_type %q", mediaType)
	}
}
