package bridgev2

import (
	"crypto/sha1"
	"encoding/hex"
	"regexp"
	"strings"

	"maunium.net/go/mautrix/bridgev2/networkid"
)

var nonIDChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func normalizeIDPart(in string) string {
	trimmed := strings.TrimSpace(in)
	if trimmed == "" {
		return "unknown"
	}
	normalized := nonIDChars.ReplaceAllString(trimmed, "_")
	normalized = strings.Trim(normalized, "_")
	if normalized == "" {
		sum := sha1.Sum([]byte(trimmed))
		return "user_" + hex.EncodeToString(sum[:])[:12]
	}
	return strings.ToLower(normalized)
}

func makeUserID(name string) networkid.UserID {
	return networkid.UserID(normalizeIDPart(name))
}

func makePortalID(chatName string) networkid.PortalID {
	return networkid.PortalID(normalizeIDPart(chatName))
}

func makeUserLoginID(label string) networkid.UserLoginID {
	return networkid.UserLoginID(normalizeIDPart(label))
}

func makeMessageID(remoteID string) networkid.MessageID {
	return networkid.MessageID(normalizeIDPart(remoteID))
}
