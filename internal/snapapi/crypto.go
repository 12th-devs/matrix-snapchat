package snapapi

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"net/url"
	"strings"
	"time"

	snapcrypto "github.com/0xzer/snapper/crypto"
	"github.com/0xzer/snapper/protos"
)

func decryptEELGCM(data, key, nonce []byte) ([]byte, bool) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, false
	}
	if len(data) == 0 || len(nonce) == 0 {
		return nil, false
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, false
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, false
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, false
	}
	decrypted, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, false
	}
	return decrypted, true
}

func encodeUUIDString(value string) (*protos.UUID, error) {
	encoded, err := snapcrypto.EncodeUUID(value)
	if err != nil {
		return nil, err
	}
	return &protos.UUID{EncodedId: encoded}, nil
}

func uuidToString(value *protos.UUID) string {
	if value == nil || len(value.GetEncodedId()) != 16 {
		return ""
	}
	decoded, err := snapcrypto.DecodeUUID(value.GetEncodedId())
	if err != nil {
		return ""
	}
	return decoded
}

func frameGRPC(message []byte) []byte {
	var buf bytes.Buffer
	buf.WriteByte(0)
	_ = binary.Write(&buf, binary.BigEndian, int32(len(message)))
	buf.Write(message)
	return buf.Bytes()
}

func randomUint64() uint64 {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return uint64(time.Now().UnixNano())
	}
	return binary.BigEndian.Uint64(data[:])
}

func timeFromMillis(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(value)
}

func shortID(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Snapchat"
	}
	if len(value) <= 8 {
		return value
	}
	return value[:8]
}

func extractCookieValue(cookieString, key string) string {
	for _, rawPart := range strings.Split(cookieString, ";") {
		part := strings.TrimSpace(rawPart)
		if part == "" {
			continue
		}
		name, value, ok := strings.Cut(part, "=")
		if ok && strings.TrimSpace(name) == key {
			unescaped, err := url.QueryUnescape(value)
			if err == nil {
				return unescaped
			}
			return value
		}
	}
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
