package snapapi

import (
	"strings"
	"time"

	"github.com/0xzer/snapper/protos"
)

func (c *Client) retentionDurationForChat(chatID string) time.Duration {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return 0
	}
	c.mu.Lock()
	conv := c.conversations[chatID]
	c.mu.Unlock()
	return retentionDurationFromConversation(conv)
}

func (c *Client) RetentionDuration(chatID string) time.Duration {
	return c.retentionDurationForChat(chatID)
}

func (c *Client) RetentionDurationKnown(chatID string) (time.Duration, bool) {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return 0, false
	}
	c.mu.Lock()
	conv := c.conversations[chatID]
	c.mu.Unlock()
	if conv == nil {
		return 0, false
	}
	return retentionDurationFromConversation(conv), true
}

func retentionDurationFromConversation(conv *protos.Conversation) time.Duration {
	if conv == nil {
		return 0
	}
	dynamic := conv.GetRetentionPolicy().GetDynamic()
	if dynamic == nil {
		return 0
	}
	seconds := dynamic.GetReadRetentionTimeSeconds()
	if seconds <= 0 {
		seconds = dynamic.GetUnreadRetentionTimeSeconds()
	}
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
