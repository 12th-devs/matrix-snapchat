package connector

import (
	"strings"
	"time"

	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"maunium.net/go/mautrix/bridgev2/database"
	"maunium.net/go/mautrix/event"
)

func disappearingSetting(seconds int64) database.DisappearingSetting {
	if seconds <= 0 {
		return database.DisappearingSetting{}
	}
	return database.DisappearingSetting{
		Type:  event.DisappearingTypeAfterSend,
		Timer: time.Duration(seconds) * time.Second,
	}
}

func (sa *SnapchatAPI) disappearingSettingForChat(chatID string) *database.DisappearingSetting {
	chatID = strings.TrimSpace(chatID)
	if chatID == "" {
		return nil
	}
	sa.mu.Lock()
	seconds := sa.chatDisappearAfter[chatID]
	sa.mu.Unlock()
	setting := disappearingSetting(seconds)
	if setting.Type == "" {
		return nil
	}
	return &setting
}

func (sa *SnapchatAPI) disappearingSettingForMessage(chatID string, message sidecar.Message) database.DisappearingSetting {
	if message.Saved {
		return database.DisappearingSetting{}
	}
	if message.DisappearAfterSeconds > 0 {
		return disappearingSetting(message.DisappearAfterSeconds)
	}
	if chatSetting := sa.disappearingSettingForChat(chatID); chatSetting != nil {
		return *chatSetting
	}
	return database.DisappearingSetting{}
}
