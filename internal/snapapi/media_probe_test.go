package snapapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
)

// TestMediaEnvelopeProbe is an opt-in live diagnostic: it reports the
// ContentEnvelope structure (no keys, cookies, tokens, or signed URLs) for
// media-bearing messages so the real media location can be identified.
// Skips unless SNAPCHAT_PROBE_CONNECTOR/SECRET/CHAT are set.
func TestMediaEnvelopeProbe(t *testing.T) {
	base := strings.TrimSpace(os.Getenv("SNAPCHAT_PROBE_CONNECTOR"))
	secret := strings.TrimSpace(os.Getenv("SNAPCHAT_PROBE_SECRET"))
	chatID := strings.TrimSpace(os.Getenv("SNAPCHAT_PROBE_CHAT"))
	target := strings.TrimSpace(os.Getenv("SNAPCHAT_PROBE_MESSAGE"))
	if base == "" || secret == "" || chatID == "" {
		t.Skip("probe env not set")
	}

	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(base, "/")+"/session/api-auth", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("X-Bridge-Secret", secret)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var auth struct {
		State               string `json:"state"`
		Authenticated       bool   `json:"authenticated"`
		CookieString        string `json:"cookieString"`
		SSOToken            string `json:"ssoToken"`
		SelfUserID          string `json:"selfUserID"`
		BrowserUserAgent    string `json:"browserUserAgent"`
		SnapClientUserAgent string `json:"snapClientUserAgent"`
		SecChUA             string `json:"secChUa"`
		SecChUAPlatform     string `json:"secChUaPlatform"`
		GRPCWebUserAgent    string `json:"grpcWebUserAgent"`
		MCSCOFIDsBin        string `json:"mcsCofIdsBin"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}
	if !auth.Authenticated || auth.CookieString == "" {
		t.Fatalf("connector session not ready: state=%s authenticated=%t", auth.State, auth.Authenticated)
	}

	client, err := New(Config{
		CookieString:        auth.CookieString,
		SSOToken:            auth.SSOToken,
		SelfUserID:          auth.SelfUserID,
		UserAgent:           auth.BrowserUserAgent,
		SnapClientUserAgent: auth.SnapClientUserAgent,
		SecChUA:             auth.SecChUA,
		SecChUAPlatform:     auth.SecChUAPlatform,
		GRPCWebUserAgent:    auth.GRPCWebUserAgent,
		MCSCOFIDsBin:        auth.MCSCOFIDsBin,
		Timeout:             45 * time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := client.Authenticate(ctx); err != nil {
		t.Fatal(err)
	}

	conv, err := client.conversation(ctx, chatID)
	if err != nil {
		t.Fatal(err)
	}
	requestedCount := 40
	if raw := strings.TrimSpace(os.Getenv("SNAPCHAT_PROBE_COUNT")); raw != "" {
		if parsed, parseErr := strconv.Atoi(raw); parseErr == nil && parsed > 0 && parsed <= 500 {
			requestedCount = parsed
		}
	}
	qreq := &protos.QueryMessagesRequest{
		SelfUserId:         client.selfUUID(),
		ConversationId:     conv.GetConversationId(),
		CurrentVersion:     conv.GetVersion(),
		RequestedCountSize: int32(requestedCount),
	}
	var qresp protos.QueryMessagesResponse
	if err := client.doGRPC(ctx, paths.QUERY_MESSAGES, qreq, &qresp); err != nil {
		t.Fatal(err)
	}

	dumped := 0
	for _, msg := range qresp.GetMessages() {
		id := fmt.Sprintf("%d", msg.GetMessageId())
		if target != "" && id != target {
			continue
		}
		envelope := msg.GetContents()
		if envelope == nil {
			continue
		}
		hasMedia := len(envelope.GetRemoteMediaInfos()) > 0 || len(envelope.GetMediaReferenceLists()) > 0 || len(envelope.GetContents()) > 0
		if !hasMedia {
			continue
		}
		dumped++
		if dumped > 10 {
			break
		}
		t.Logf("message_id=%s content_type=%s", id, envelope.GetContentType())
		enc := envelope.GetEnvelopeEncryption()
		t.Logf("  encryption: method=%T clearTextMediaKey=%T key_len=%d iv_len=%d clearTextEel=%T eel=%T",
			enc.GetMethod(), enc.GetClearTextMediaKey(), len(enc.GetClearTextMediaKey().GetMediaKey()), len(enc.GetClearTextMediaKey().GetMediaIv()),
			enc.GetClearTextEelKeyEncryption(), enc.GetEelEncryption())
		contents := envelope.GetContents()
		t.Logf("  contents: len=%d", len(contents))
		decoded := client.envelopeContentsForDecode(ctx, chatID, id, envelope, 0)
		t.Logf("  decoded contents: len=%d", len(decoded))
		walkProtoFields(t, "  decoded", decoded, 0, 6)
		if envelope.GetContentType() == protos.ContentType_EXTERNAL_MEDIA {
			key, iv, err := ExternalMediaEncryptionKeys(decoded)
			t.Logf("  external media key_len=%d iv_len=%d metadata_error=%v", len(key), len(iv), err)
			if os.Getenv("SNAPCHAT_PROBE_WEB_DECRYPT") == "true" && len(key) == 32 && len(iv) == 16 {
				// Narrowest in-memory handoff: the decoded key/IV only cross to
				// the connector's existing debug decrypt probe over loopback,
				// are never logged here, and are not part of any response body
				// the connector returns.
				descriptorHex := strings.TrimSpace(os.Getenv("SNAPCHAT_PROBE_DESCRIPTOR_HEX"))
				if descriptorHex == "" {
					t.Log("  web decrypt skipped: SNAPCHAT_PROBE_DESCRIPTOR_HEX not set")
				} else {
					body := map[string]string{
						"descriptorHex": descriptorHex,
						"chatId":        chatID,
						"messageId":     id,
						"mediaKeyB64":   base64.StdEncoding.EncodeToString(key),
						"mediaIVB64":    base64.StdEncoding.EncodeToString(iv),
					}
					payload, marshalErr := json.Marshal(body)
					if marshalErr != nil {
						t.Fatalf("  web decrypt body marshal: %v", marshalErr)
					}
					decryptReq, reqErr := http.NewRequest(http.MethodPost, strings.TrimRight(base, "/")+"/debug/media-decrypt", bytes.NewReader(payload))
					if reqErr != nil {
						t.Fatalf("  web decrypt request: %v", reqErr)
					}
					decryptReq.Header.Set("X-Bridge-Secret", secret)
					decryptReq.Header.Set("Content-Type", "application/json")
					decryptResp, postErr := http.DefaultClient.Do(decryptReq)
					if postErr != nil {
						t.Fatalf("  web decrypt call: %v", postErr)
					}
					decryptBody, _ := io.ReadAll(io.LimitReader(decryptResp.Body, 1<<20))
					decryptResp.Body.Close()
					t.Logf("  web decrypt http_status=%d response=%s", decryptResp.StatusCode, string(decryptBody))
				}
			}
			if os.Getenv("SNAPCHAT_PROBE_DOWNLOAD") == "true" {
				parsed := client.messageFromProto(ctx, chatID, msg)
				for _, attachment := range parsed.Media {
					data, mime, info, err := client.DownloadMediaWithInfo(ctx, attachment)
					t.Logf("  download: descriptor_id=%q bytes=%d mime=%s info=%+v error=%v", MediaDescriptorID(attachment.Data), len(data), mime, info, err)
					if err != nil {
						t.Error("ordinary media download failed")
					}
				}
			}
		}
		for i, info := range envelope.GetRemoteMediaInfos() {
			obj := info.GetContentObject()
			t.Logf("  remoteMedia[%d]: mediaType=%v hasAudio=%t contentObject_len=%d url_present=%t descriptor_id=%q",
				i, info.GetMediaType(), info.GetHasAudio(), len(obj), info.GetContentUrl() != "", MediaDescriptorID(obj))
		}
		for li, list := range envelope.GetMediaReferenceLists() {
			for ri, ref := range list.GetReference() {
				obj := ref.GetContentObject()
				t.Logf("  mediaRef[%d][%d]: mediaType=%v url_present=%t contentObject_len=%d mediaListId=%d descriptor_id=%q descriptor_hex=%x",
					li, ri, ref.GetMediaType(), ref.GetUrl() != "", len(obj), ref.GetMediaListId(), MediaDescriptorID(obj), obj)
				describeProtoShape(t, "  mediaRef.obj", obj)
			}
		}
		if thumbs := envelope.GetThumbnails().GetThumbnails(); len(thumbs) > 0 {
			for i, thumb := range thumbs {
				t.Logf("  thumbnail[%d]: mediaListId=%d mediaRefIndex=%d", i, thumb.GetMediaId().GetMediaListId(), thumb.GetMediaReferenceListIndex())
			}
		}
		walkProtoFields(t, "  contents", contents, 0, 2)
	}
	if dumped == 0 {
		t.Log("no media-bearing messages matched")
	}
}

// describeProtoShape reports field numbers and wire types of a small opaque
// descriptor so its structure can be identified without printing key material.
func describeProtoShape(t *testing.T, prefix string, data []byte) {
	for offset := 0; offset < len(data); {
		tag, next, ok := readProtoVarint(data, offset)
		if !ok {
			t.Logf("%s: not valid protobuf at offset=%d (first16=%x)", prefix, offset, data[:minInt(16, len(data))])
			return
		}
		offset = next
		fieldNumber, wireType := int(tag>>3), int(tag&0x7)
		switch wireType {
		case 0:
			value, next, ok := readProtoVarint(data, offset)
			if !ok {
				return
			}
			offset = next
			t.Logf("%s field=%d varint=%d", prefix, fieldNumber, value)
		case 2:
			length, next, ok := readProtoVarint(data, offset)
			if !ok || length > uint64(len(data)-next) {
				t.Logf("%s field=%d bytes_len=truncated", prefix, fieldNumber)
				return
			}
			start, end := next, next+int(length)
			offset = end
			b := data[start:end]
			printable := true
			for _, c := range b {
				if c < 0x21 || c > 0x7e {
					printable = false
					break
				}
			}
			t.Logf("%s field=%d bytes_len=%d printable=%t value=%s", prefix, fieldNumber, len(b), printable, string(b))
		default:
			t.Logf("%s field=%d wire_type=%d", prefix, fieldNumber, wireType)
			return
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// walkProtoFields prints a shallow generic proto tree, highlighting any
// length-delimited field that looks like a URL (truncated for safety).
func walkProtoFields(t *testing.T, prefix string, data []byte, depth, maxDepth int) {
	if depth > maxDepth || len(data) == 0 {
		return
	}
	for offset := 0; offset < len(data); {
		tag, next, ok := readProtoVarint(data, offset)
		if !ok {
			return
		}
		offset = next
		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)
		switch wireType {
		case 0:
			value, next, ok := readProtoVarint(data, offset)
			if !ok {
				return
			}
			offset = next
			t.Logf("%s field=%d varint=%d", prefix, fieldNumber, value)
		case 1:
			if offset+8 > len(data) {
				return
			}
			offset += 8
		case 5:
			if offset+4 > len(data) {
				return
			}
			offset += 4
		case 2:
			length, next, ok := readProtoVarint(data, offset)
			if !ok || length > uint64(len(data)-next) {
				return
			}
			start := next
			end := start + int(length)
			offset = end
			b := data[start:end]
			t.Logf("%s field=%d bytes_len=%d", prefix, fieldNumber, len(b))
			if fieldNumber == 3 || fieldNumber == 9 || fieldNumber == 17 || fieldNumber == 6 {
				walkProtoFields(t, prefix+".f"+fmt.Sprint(fieldNumber), b, depth+1, maxDepth)
			}
		default:
			return
		}
	}
}
