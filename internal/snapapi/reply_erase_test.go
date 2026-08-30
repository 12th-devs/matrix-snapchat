package snapapi

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/0xzer/snapper/data/methods"
	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
)

// TestSendTextWithReplyAttachesQuote verifies that a reply target is sent as a
// real Snapchat reply: CreateContentMessage.FeatureAttachment carries a
// ReplyMessageInfo referencing the quoted message ID (the same shape the web
// client sends), not a copied-text fake.
func TestSendTextWithReplyAttachesQuote(t *testing.T) {
	const (
		chatID  = "aaaaaaaa-0000-0000-0000-000000000001"
		quoted  = int64(777)
		version = int64(42)
	)
	var createRequests []*protos.CreateContentMessageRequest
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		switch r.URL.Path {
		case urlPath(paths.DELTA_SYNC_CONVERSATIONS):
			frame, err2 := methods.ReadResponseFrame(body)
			if err2 != nil {
				t.Fatalf("delta frame unreadable: %v", err2)
			}
			var req protos.DeltaSyncRequest
			if err2 := protos.DecodeProtoMessage(frame, &req); err2 != nil {
				t.Fatalf("delta proto undecodable: %v", err2)
			}
			payload, _ := protos.EncodeProtoMessage(&protos.DeltaSyncResponse{
				Metadata: &protos.DeltaSyncResponse_Conversation{
					Conversation: &protos.Conversation{
						ConversationId: req.GetConversationId(),
						Version:        version,
					},
				},
			})
			w.Write(grpcWebData(payload))
		case urlPath(paths.CREATE_CONTENT_MESSAGE):
			createRequests = append(createRequests, decodeCreateContentRequest(t, body))
			payload, _ := protos.EncodeProtoMessage(&protos.CreateContentMessageResponse{
				Result: []*protos.CreateContentMessageResult{{Success: true}},
			})
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTestClient(t, handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	if _, err := c.SendText(context.Background(), chatID, "a reply", quoted); err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if len(createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(createRequests))
	}
	req := createRequests[0]
	if len(req.GetFeatureAttachment()) != 1 {
		t.Fatalf("feature attachments = %d, want 1", len(req.GetFeatureAttachment()))
	}
	att := req.GetFeatureAttachment()[0]
	if got := att.GetReplyMessageInfo().GetQuotedMessageId(); got != quoted {
		t.Fatalf("QuotedMessageId = %d, want %d", got, quoted)
	}
}

// decodeCreateContentRequest decodes the framed CreateContentMessageRequest a
// doGRPC send call makes.
func decodeCreateContentRequest(t *testing.T, body []byte) *protos.CreateContentMessageRequest {
	t.Helper()
	frame, err := methods.ReadResponseFrame(body)
	if err != nil {
		t.Fatalf("create frame unreadable: %v", err)
	}
	var req protos.CreateContentMessageRequest
	if err := protos.DecodeProtoMessage(frame, &req); err != nil {
		t.Fatalf("create proto undecodable: %v", err)
	}
	return &req
}

func TestCreatedMessageIDFromConversationDestinationResult(t *testing.T) {
	const createdMessageID = uint64(9089)
	resp := &protos.CreateContentMessageResponse{
		ClientResolutionId: 475240075217556186,
		Result: []*protos.CreateContentMessageResult{{
			Success: true,
			DestinationRes: &protos.CreateContentMessageResult_ConversationDestinationResult{
				ConversationDestinationResult: &protos.ConversationDestinationResult{
					CreatedMessageId: createdMessageID,
				},
			},
		}},
	}

	if got := createdMessageIDFromResponse(resp); got != "9089" {
		t.Fatalf("createdMessageIDFromResponse = %q, want %d", got, createdMessageID)
	}
}

// TestSendTextWithoutReplyOmitsAttachment verifies a normal message carries no
// reply attachment (no faked relations).
func TestSendTextWithoutReplyHasNoAttachment(t *testing.T) {
	const chatID = "aaaaaaaa-0000-0000-0000-000000000001"
	var createRequests []*protos.CreateContentMessageRequest
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		switch r.URL.Path {
		case urlPath(paths.DELTA_SYNC_CONVERSATIONS):
			frame, _ := methods.ReadResponseFrame(body)
			var req protos.DeltaSyncRequest
			_ = protos.DecodeProtoMessage(frame, &req)
			payload, _ := protos.EncodeProtoMessage(&protos.DeltaSyncResponse{
				Metadata: &protos.DeltaSyncResponse_Conversation{
					Conversation: &protos.Conversation{
						ConversationId: req.GetConversationId(),
						Version:        42,
					},
				},
			})
			w.Write(grpcWebData(payload))
		case urlPath(paths.CREATE_CONTENT_MESSAGE):
			createRequests = append(createRequests, decodeCreateContentRequest(t, body))
			payload, _ := protos.EncodeProtoMessage(&protos.CreateContentMessageResponse{
				Result: []*protos.CreateContentMessageResult{{Success: true}},
			})
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTestClient(t, handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	if _, err := c.SendText(context.Background(), chatID, "plain message", 0); err != nil {
		t.Fatalf("SendText: %v", err)
	}
	if len(createRequests) != 1 {
		t.Fatalf("create requests = %d, want 1", len(createRequests))
	}
	if req := createRequests[0]; len(req.GetFeatureAttachment()) != 0 {
		t.Fatalf("no-reply send must not carry feature attachments, got %d", len(req.GetFeatureAttachment()))
	}
}

// TestEraseMessageSendsEraseAction verifies deletion/unsend uses the Erase
// update action with the conversation's CURRENT version.
func TestEraseMessageSendsEraseAction(t *testing.T) {
	const (
		chatID       = "aaaaaaaa-0000-0000-0000-000000000002"
		messageID    = int64(42)
		freshVersion = int64(999)
	)
	var updateRequests []*protos.UpdateContentMessageRequest
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		switch r.URL.Path {
		case urlPath(paths.DELTA_SYNC_CONVERSATIONS):
			frame, err2 := methods.ReadResponseFrame(body)
			if err2 != nil {
				t.Fatalf("delta frame unreadable: %v", err2)
			}
			var req protos.DeltaSyncRequest
			if err2 := protos.DecodeProtoMessage(frame, &req); err2 != nil {
				t.Fatalf("delta proto undecodable: %v", err2)
			}
			payload, _ := protos.EncodeProtoMessage(&protos.DeltaSyncResponse{
				Metadata: &protos.DeltaSyncResponse_Conversation{
					Conversation: &protos.Conversation{
						ConversationId: req.GetConversationId(),
						Version:        freshVersion,
					},
				},
			})
			w.Write(grpcWebData(payload))
		case urlPath(paths.UPDATE_CONTENT_MESSAGE):
			frame, err2 := methods.ReadResponseFrame(body)
			if err2 != nil {
				t.Fatalf("update frame unreadable: %v", err2)
			}
			var req protos.UpdateContentMessageRequest
			if err2 := protos.DecodeProtoMessage(frame, &req); err2 != nil {
				t.Fatalf("update proto undecodable: %v", err2)
			}
			updateRequests = append(updateRequests, &req)
			payload, _ := protos.EncodeProtoMessage(&protos.UpdateContentMessageResponse{Success: true})
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTestClient(t, handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	if err := c.EraseMessage(context.Background(), chatID, 42); err != nil {
		t.Fatalf("EraseMessage: %v", err)
	}
	if len(updateRequests) != 1 {
		t.Fatalf("update requests = %d, want 1", len(updateRequests))
	}
	req := updateRequests[0]
	if req.GetUpdate().GetErase() == nil {
		t.Fatal("Update must be an Erase action")
	}
	if req.GetUpdate().GetMessageId() != messageID {
		t.Fatalf("Update.MessageId = %d, want 42", req.GetUpdate().GetMessageId())
	}
	if req.GetCurrentVersion() != freshVersion {
		t.Fatalf("CurrentVersion = %d, want fresh %d", req.GetCurrentVersion(), freshVersion)
	}
}

// TestEraseMessageFailsWithoutExplicitSuccess verifies a delete is only
// treated as successful when Snapchat explicitly confirms Success=true. A
// retryable rejection (Success=false, Retryable=true) must surface as an error
// so the bridge never reports a destructive unsend the server did not commit.
func TestEraseMessageFailsWithoutExplicitSuccess(t *testing.T) {
	const (
		chatID       = "aaaaaaaa-0000-0000-0000-000000000003"
		freshVersion = int64(555)
	)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		switch r.URL.Path {
		case urlPath(paths.DELTA_SYNC_CONVERSATIONS):
			frame, err2 := methods.ReadResponseFrame(body)
			if err2 != nil {
				t.Fatalf("delta frame unreadable: %v", err2)
			}
			var req protos.DeltaSyncRequest
			if err2 := protos.DecodeProtoMessage(frame, &req); err2 != nil {
				t.Fatalf("delta proto undecodable: %v", err2)
			}
			payload, _ := protos.EncodeProtoMessage(&protos.DeltaSyncResponse{
				Metadata: &protos.DeltaSyncResponse_Conversation{
					Conversation: &protos.Conversation{
						ConversationId: req.GetConversationId(),
						Version:        freshVersion,
					},
				},
			})
			w.Write(grpcWebData(payload))
		case urlPath(paths.UPDATE_CONTENT_MESSAGE):
			payload, _ := protos.EncodeProtoMessage(&protos.UpdateContentMessageResponse{
				Success:   false,
				Retryable: true,
			})
			w.Write(grpcWebData(payload))
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(http.StatusForbidden)
		}
	})
	c, srv := newAuthedTestClient(t, handler)
	defer srv.Close()
	overrideListingURL(t, srv)

	if err := c.EraseMessage(context.Background(), chatID, 77); err == nil {
		t.Fatal("EraseMessage must fail when Success=false even if Retryable=true")
	}
}

// mustTextContents encodes a Snapchat text content payload.
func mustTextContents(t *testing.T, text string) []byte {
	t.Helper()
	contentBytes, err := protos.EncodeProtoMessage(&protos.Contents{
		Content: &protos.Contents_Text{
			Text: &protos.Text{Text: text},
		},
	})
	if err != nil {
		t.Fatalf("encode text contents: %v", err)
	}
	return contentBytes
}

// TestMessageFromProtoReplyAndTombstone verifies the reply reference and the
// tombstone (erased) flag are parsed from MessageMetadata.
func TestMessageFromProtoReplyAndTombstone(t *testing.T) {
	msg := (&Client{}).messageFromProto(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 55,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    mustTextContents(t, "reply text"),
		},
		MetaData: &protos.MessageMetadata{
			QuotedMessageInfo: &protos.MessageMetadata_QuotedMetadata{
				QuotedMetadata: &protos.MessageMetadata_QuotedMessageMetadata{QuotedMessageId: 123},
			},
		},
	})
	if msg.QuotedMessageID != 123 {
		t.Fatalf("QuotedMessageID = %d, want 123", msg.QuotedMessageID)
	}
	if msg.Tombstone {
		t.Fatal("normal message must not be marked tombstone")
	}
}

// TestMessageFromProtoTombstone verifies erased messages are flagged instead of
// being displayed.
func TestMessageFromProtoTombstone(t *testing.T) {
	msg := (&Client{}).messageFromProto(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 77,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    mustTextContents(t, "gone"),
		},
		MetaData: &protos.MessageMetadata{Tombstone: true},
	})
	if !msg.Tombstone {
		t.Fatal("tombstone message was not flagged")
	}
}

func TestMessageFromProtoNonTombstone(t *testing.T) {
	msg := (&Client{}).messageFromProto(context.Background(), "chat-1", &protos.ContentMessage{
		MessageId: 78,
		Contents: &protos.ContentEnvelope{
			ContentType: protos.ContentType_CHAT,
			Contents:    mustTextContents(t, "hello"),
		},
		MetaData: &protos.MessageMetadata{},
	})
	if msg.Tombstone {
		t.Fatal("normal message marked as tombstone")
	}
}
