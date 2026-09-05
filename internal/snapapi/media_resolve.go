package snapapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/0xzer/snapper/data/headers"
	"github.com/0xzer/snapper/data/paths"
	"google.golang.org/protobuf/encoding/protowire"
)

// ExternalMediaEncryptionKeys extracts the regular image-DM metadata path
// encoded by snapcap-native/src/api/_media_send.ts: encodeCreateContentMessageMedia.
// The pinned Snapper Contents proto comments out ExternalMedia entirely.
// It intentionally rejects multiple items: associating keys with attachments
// requires a verified media-reference mapping, not positional guesswork.
func ExternalMediaEncryptionKeys(contents []byte) (key, iv []byte, err error) {
	current := contents
	for _, field := range []protowire.Number{3, 3, 5, 1, 1} {
		parts, parseErr := mediaProtoField(current, field)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		if len(parts) == 0 {
			return nil, nil, nil
		}
		if len(parts) != 1 {
			return nil, nil, fmt.Errorf("ambiguous ExternalMedia metadata")
		}
		current = parts[0]
	}
	var foundKey, foundIV []byte
	for _, field := range []protowire.Number{19, 4} {
		parts, parseErr := mediaProtoField(current, field)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		if len(parts) == 0 {
			continue
		}
		if len(parts) != 1 {
			return nil, nil, fmt.Errorf("ambiguous ExternalMedia encryption")
		}
		keys, parseErr := mediaProtoField(parts[0], 1)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		ivs, parseErr := mediaProtoField(parts[0], 2)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		if len(keys) != 1 || len(ivs) != 1 {
			return nil, nil, fmt.Errorf("incomplete ExternalMedia encryption")
		}
		k, v := keys[0], ivs[0]
		if field == 4 {
			k, parseErr = base64.StdEncoding.DecodeString(string(k))
			if parseErr != nil {
				return nil, nil, fmt.Errorf("invalid ExternalMedia base64 key")
			}
			v, parseErr = base64.StdEncoding.DecodeString(string(v))
			if parseErr != nil {
				return nil, nil, fmt.Errorf("invalid ExternalMedia base64 IV")
			}
		}
		if len(k) != 32 || len(v) != 16 {
			return nil, nil, fmt.Errorf("invalid ExternalMedia key or IV length")
		}
		if foundKey != nil && (!bytes.Equal(foundKey, k) || !bytes.Equal(foundIV, v)) {
			return nil, nil, fmt.Errorf("conflicting ExternalMedia encryption metadata")
		}
		foundKey, foundIV = append([]byte(nil), k...), append([]byte(nil), v...)
	}
	return foundKey, foundIV, nil
}

// The wire schema is from the cached official web bundle
// .codex-build/snapcap-native/vendor/snap-bundle/cf-st.sc-cdn.net/dw/9846a7958a5f0bee7197.js:
// MediaDeliveryService.resolveContentObjects, On/Dn/Mn/Ln/Fn/xn/Bn.
// Snapper's pinned generated types do not include this service.
// This resolves content references only; it does not mark messages viewed.
// Callers must gate this on ordinary attachments, never view-once SNAPs.
func (c *Client) ResolveMediaDescriptor(ctx context.Context, descriptor []byte) ([]string, error) {
	if MediaDescriptorID(descriptor) == "" {
		return nil, fmt.Errorf("media resolver requires a recognized content descriptor")
	}
	if c.http == nil || c.cookies == nil || c.tokens == nil {
		return nil, fmt.Errorf("media descriptor resolver has no authenticated client")
	}
	// requests[0].reference.contentObject = the descriptor bytes. The captured
	// Snapchat Web bundle (a50bef017778a089ee5f.js: Dn/rt/je definitions) shows
	// the descriptor IS a serialized v2 ContentObject: field 1 holds
	// contentObjectId and field 2 wraps a ContentDescriptor whose field 2 is
	// contentId. The web client sends it parsed via the contentObject oneof
	// case (Dn field 1); sending the same bytes via v2ContentObject (field 3)
	// returns an empty HTTP 200 body, so the raw bytes are passed through
	// verbatim here under the supported field.
	request := mediaProtoBytes(1, mediaProtoBytes(1, descriptor))
	framed := frameGRPC(request)
	header := headers.NewCoreHeaders(c.cookies, c.tokens, c.device)
	c.applyUserAgentHeaders(header)
	c.applyBrowserGRPCHeaders(header)
	// Snapchat Web now runs on www.snapchat.com and calls web.snapchat.com
	// cross-origin, so the browser sends this Origin/Referer pair and
	// sec-fetch metadata on every content-service RPC. The content gateway
	// rejects requests without it (empty 200), unlike the messaging service.
	header.Set("origin", "https://www.snapchat.com")
	header.Set("referer", "https://www.snapchat.com/")
	header.Set("sec-fetch-dest", "empty")
	header.Set("sec-fetch-mode", "cors")
	header.Set("sec-fetch-site", "cross-site")
	body, meta, err := c.doHTTPDetailed(ctx, paths.WEB_BASE_URL+"/snapchat.content.v2.MediaDeliveryService/resolveContentObjects", http.MethodPost, header, framed)
	if err != nil {
		return nil, fmt.Errorf("media resolver request framed_len=%d payload_len=%d: %w (%s)", len(framed), len(request), err, meta)
	}
	payload, err := mediaGRPCPayload(body)
	if err != nil {
		return nil, fmt.Errorf("%w (response frames: %s; rpc %s)", err, mediaRPCFrameDiagnostics(body), meta)
	}
	urls, err := resolvedMediaURLs(payload)
	if err != nil {
		return nil, fmt.Errorf("%w (payload: %s; rpc %s)", err, mediaPayloadDiagnostics(payload), meta)
	}
	return urls, nil
}

// mediaRPCFrameDiagnostics reports gRPC-web frame flags and sizes only, so a
// failing resolveContentObjects response can be identified without exposing
// any URL or token material.
func mediaRPCFrameDiagnostics(body []byte) string {
	var parts []string
	for len(body) > 0 {
		if len(body) < 5 {
			parts = append(parts, fmt.Sprintf("trailer_bytes=%d", len(body)))
			break
		}
		flags := body[0]
		size := binary.BigEndian.Uint32(body[1:5])
		if uint64(size) > uint64(len(body)-5) {
			parts = append(parts, fmt.Sprintf("frame flags=%d size=%d truncated", flags, size))
			break
		}
		parts = append(parts, fmt.Sprintf("frame flags=%d size=%d", flags, size))
		body = body[5+int(size):]
	}
	if len(parts) == 0 {
		return "empty"
	}
	return strings.Join(parts, ", ")
}

// mediaPayloadDiagnostics reports the field structure (field numbers, wire
// types, lengths, and HTTPS URL hostnames only) of a resolver response payload.
// String values are never included so signed URLs or tokens cannot leak.
func mediaPayloadDiagnostics(payload []byte) string {
	var sb strings.Builder
	var walk func(data []byte, prefix string, depth int)
	walk = func(data []byte, prefix string, depth int) {
		if depth > 6 || len(data) == 0 {
			return
		}
		for len(data) > 0 {
			number, kind, n := protowire.ConsumeTag(data)
			if n < 0 {
				sb.WriteString(prefix + " <malformed>")
				return
			}
			data = data[n:]
			switch kind {
			case protowire.BytesType:
				value, n := protowire.ConsumeBytes(data)
				if n < 0 {
					sb.WriteString(prefix + " <malformed>")
					return
				}
				data = data[n:]
				entry := fmt.Sprintf(" %s%d:bytes_len=%d", prefix, number, len(value))
				if u, err := url.Parse(string(value)); err == nil && u.Scheme == "https" && u.Hostname() != "" {
					entry += fmt.Sprintf(" https_url_host=%s", u.Hostname())
				} else if printableProtoText(value) {
					entry += fmt.Sprintf(" ascii_len=%d", len(value))
				}
				sb.WriteString(entry)
				walk(value, prefix+fmt.Sprintf(".f%d", number), depth+1)
			case protowire.VarintType:
				v, n := protowire.ConsumeVarint(data)
				if n < 0 {
					sb.WriteString(prefix + " <malformed>")
					return
				}
				data = data[n:]
				sb.WriteString(fmt.Sprintf(" %s%d:varint=%d", prefix, number, v))
			default:
				sb.WriteString(fmt.Sprintf(" %s%d:wire=%d", prefix, number, kind))
				return
			}
		}
	}
	walk(payload, "f", 0)
	return strings.TrimSpace(sb.String())
}

func printableProtoText(value []byte) bool {
	if len(value) == 0 {
		return false
	}
	for _, b := range value {
		if b < 0x21 || b > 0x7e {
			return false
		}
	}
	return true
}

func mediaProtoBytes(field protowire.Number, value []byte) []byte {
	return protowire.AppendBytes(protowire.AppendTag(nil, field, protowire.BytesType), value)
}

// Check every frame, including error trailers after a valid data frame.
func mediaGRPCPayload(body []byte) ([]byte, error) {
	var payload []byte
	for len(body) > 0 {
		if len(body) < 5 {
			return nil, fmt.Errorf("truncated media RPC frame")
		}
		flags, size := body[0], uint64(binary.BigEndian.Uint32(body[1:5]))
		body = body[5:]
		if size > uint64(len(body)) {
			return nil, fmt.Errorf("truncated media RPC payload")
		}
		frame := body[:int(size)]
		body = body[int(size):]
		switch flags {
		case 0:
			if payload != nil {
				return nil, fmt.Errorf("multiple media RPC data frames")
			}
			payload = append([]byte{}, frame...)
		case 0x80:
			for _, line := range strings.Split(string(frame), "\n") {
				key, value, ok := strings.Cut(strings.TrimSpace(line), ":")
				if ok && key == "grpc-status" && strings.TrimSpace(value) != "0" {
					return nil, fmt.Errorf("media resolver grpc status %s", strings.TrimSpace(value))
				}
			}
		default:
			return nil, fmt.Errorf("unsupported media RPC frame flags %d", flags)
		}
	}
	if payload == nil {
		return nil, fmt.Errorf("media RPC returned no data")
	}
	return payload, nil
}

// Decode only verified length-delimited fields; malformed protobuf is fatal.
func mediaProtoField(data []byte, wanted protowire.Number) ([][]byte, error) {
	var result [][]byte
	for len(data) > 0 {
		number, kind, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		data = data[n:]
		if number == wanted {
			if kind != protowire.BytesType {
				return nil, fmt.Errorf("media RPC field %d has wrong wire type", number)
			}
			value, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return nil, protowire.ParseError(n)
			}
			result = append(result, value)
		}
		n = protowire.ConsumeFieldValue(number, kind, data)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		data = data[n:]
	}
	return result, nil
}

func resolvedMediaURLs(payload []byte) ([]string, error) {
	responses, err := mediaProtoField(payload, 1)
	if err != nil {
		return nil, err
	}
	if len(responses) != 1 {
		return nil, fmt.Errorf("media resolver returned %d responses for one descriptor", len(responses))
	}
	errors, err := mediaProtoField(responses[0], 2)
	if err != nil {
		return nil, err
	}
	if len(errors) != 0 {
		return nil, fmt.Errorf("media resolver rejected content object")
	}
	current, err := mediaProtoField(responses[0], 1) // resolvedContentObject
	if err != nil {
		return nil, err
	}
	if len(current) != 1 {
		return nil, fmt.Errorf("media resolver missing resolved content object")
	}
	// Fn.resolvedContents -> xn.resolvedUrls -> Bn.url.
	for depth := 0; depth < 3; depth++ {
		var next [][]byte
		for _, item := range current {
			fields, err := mediaProtoField(item, 1)
			if err != nil {
				return nil, err
			}
			next = append(next, fields...)
		}
		current = next
	}
	var result []string
	seen := map[string]bool{}
	for _, raw := range current {
		value := string(raw)
		u, err := url.Parse(value)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return nil, fmt.Errorf("media resolver returned an invalid HTTPS download URL")
		}
		if !seen[value] {
			result = append(result, value)
			seen[value] = true
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("media resolver returned no download URLs")
	}
	return result, nil
}
