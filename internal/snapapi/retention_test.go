package snapapi

import (
	"testing"
	"time"

	"github.com/0xzer/snapper/protos"
)

func TestRetentionDurationFromConversationUsesReadTimer(t *testing.T) {
	got := retentionDurationFromConversation(&protos.Conversation{
		RetentionPolicy: &protos.RetentionPolicy{
			Policy: &protos.RetentionPolicy_Dynamic{
				Dynamic: &protos.DynamicRetentionPolicy{
					UnreadRetentionTimeSeconds: 60,
					ReadRetentionTimeSeconds:   24 * 60 * 60,
				},
			},
		},
	})
	if got != 24*time.Hour {
		t.Fatalf("retention duration = %s, want 24h", got)
	}
}

func TestRetentionDurationFromConversationFallsBackToUnreadTimer(t *testing.T) {
	got := retentionDurationFromConversation(&protos.Conversation{
		RetentionPolicy: &protos.RetentionPolicy{
			Policy: &protos.RetentionPolicy_Dynamic{
				Dynamic: &protos.DynamicRetentionPolicy{
					UnreadRetentionTimeSeconds: 60,
				},
			},
		},
	})
	if got != time.Minute {
		t.Fatalf("retention duration = %s, want 1m", got)
	}
}

func TestRetentionDurationKnownReportsCachedConversation(t *testing.T) {
	client := &Client{conversations: map[string]*protos.Conversation{
		"chat-1": {
			RetentionPolicy: &protos.RetentionPolicy{
				Policy: &protos.RetentionPolicy_Dynamic{
					Dynamic: &protos.DynamicRetentionPolicy{
						ReadRetentionTimeSeconds: 24 * 60 * 60,
					},
				},
			},
		},
		"chat-2": {},
	}}
	if got, ok := client.RetentionDurationKnown("chat-1"); !ok || got != 24*time.Hour {
		t.Fatalf("known retention = %s %v, want 24h true", got, ok)
	}
	if got, ok := client.RetentionDurationKnown("chat-2"); !ok || got != 0 {
		t.Fatalf("known no-retention = %s %v, want 0 true", got, ok)
	}
	if got, ok := client.RetentionDurationKnown("missing"); ok || got != 0 {
		t.Fatalf("missing retention = %s %v, want 0 false", got, ok)
	}
}

func TestMessageIsSaved(t *testing.T) {
	if !messageIsSaved(&protos.ContentMessage{
		Contents: &protos.ContentEnvelope{
			SavePolicy: protos.ContentEnvelope_SavePolicy_LIFETIME,
		},
	}) {
		t.Fatal("LIFETIME envelope should be treated as saved")
	}
	if messageIsSaved(&protos.ContentMessage{
		Contents: &protos.ContentEnvelope{
			SavePolicy: protos.ContentEnvelope_SavePolicy_ENVELOPE_UNSET,
		},
	}) {
		t.Fatal("unset save policy should not be treated as saved")
	}
}
