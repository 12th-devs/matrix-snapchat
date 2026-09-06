package snapapi

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/0xzer/snapper/protos"
	"google.golang.org/protobuf/encoding/protowire"
)

type MediaDownloadInfo struct {
	Source          string
	DecryptPath     string
	Bytes           int
	ContentType     string
	MediaID         string
	DescriptorShape string
	CiphertextBytes int
	// CDNFallbackReason preserves why the direct CDN attempt failed when
	// DownloadMediaWithInfo falls back to the resolver RPC. Empty when no
	// CDN attempt was made.
	CDNFallbackReason string
}

// MediaDescriptor shapes observed in live Snapchat Web envelopes. The opaque
// payload is a serialized v2 ContentObject (bundle definitions rt/je):
//   - field1-direct: field 1 = contentObjectId (ASCII), field 2 = nested
//     ContentDescriptor (field 2 = contentId ASCII, field 10 = use case) —
//     live 61-byte capture from message 745.
//   - field2-nested: field 2 = contentDescriptor whose field 2 = contentId
//     (ASCII), plus readyLocationIds (f6, repeated varint), claimId (f9),
//     useCase (f10), hostPatternVersion (f12) — live 34-byte captures from
//     messages 4114 (locations [3]) and 79 (locations [79]).
//
// Snapchat Web resolves both shapes with its in-process bolt
// content_resolution_ContentResolver, which maps readyLocationIds through the
// public bolt network mapping to deterministic CDN URLs: cf-st.sc-cdn.net
// /c/{contentId}?uc={use case} for location 3 (4114), /d/{nested contentId}
// for location 4 (745), and bolt-gcdn.sc-cdn.net /bq/{contentId}?uc={use
// case} for location 79 (79). The resolveContentObjects RPC is
// PERMISSION_DENIED for ordinary web sessions, so it must never be relied on.
type MediaDescriptor struct {
	Detected        bool
	Shape           string
	ContentObjectID string
	// CDNObjectID is the content ID the CDN URL path must carry when it
	// differs from ContentObjectID (field1-direct: the nested
	// ContentDescriptor contentId, proven by live message 745).
	CDNObjectID string
	// CDNPathSegment is the proven URL path segment for this shape:
	// "c" for field2-nested, "d" for field1-direct.
	CDNPathSegment string
	// UseCase is the descriptor field-10 varint (Web resolver maps it to the
	// CDN `uc` query parameter; absent maps to 0).
	UseCase uint64
	// ReadyLocationIDs are the ContentDescriptor readyLocationIds (field 6,
	// repeated varint): indexes into the public bolt network mapping that
	// select the serving bucket and CDN path prefix. Live values: message
	// 4114 carries [3] and message 79 carries [79]. Not a constant: the
	// earlier f6=={3} equality was an over-fit to 4114.
	ReadyLocationIDs []uint64
	// CDNEligible records that the descriptor carries the content ID and use
	// case the Web bolt resolver needs to derive a deterministic CDN URL
	// locally. Anything else must fall back to the resolver RPC.
	CDNEligible bool
}

const (
	mediaDescriptorShapeField1Direct = "field1-direct"
	mediaDescriptorShapeField2Nested = "field2-nested"
)

// mediaDescriptorUseCaseField is the descriptor field the Web content resolver
// maps to the CDN `uc` query parameter.
const mediaDescriptorUseCaseField = 10

// CDN retrieval for ordinary chat media mirrors the proven Web bolt content
// resolver output: https://cf-st.sc-cdn.net/c/{id}?uc={field10} for the
// field2-nested shape (live message 4114) and
// https://cf-st.sc-cdn.net/d/{nested contentId}?uc={field10} for the
// field1-direct shape (live message 745, verified byte-for-byte against the
// Web session: HTTP 200, no signing, no auth). The resolveContentObjects RPC
// is PERMISSION_DENIED for ordinary web sessions and stays only as a
// last-resort fallback that fails loudly.
const (
	mediaCDNHost         = "cf-st.sc-cdn.net"
	mediaCDNUseCaseParam = "uc"
)

// Proven CDN URL path segments per descriptor shape (see MediaDescriptor).
const (
	mediaCDNPathSegmentContent = "c"
	mediaCDNPathSegmentDirect  = "d"
)

const (
	maxMediaDescriptorBytes    = 512
	minMediaDescriptorIDBytes  = 8
	maxMediaDescriptorIDBytes  = 96
	maxMediaDescriptorFieldNum = 15
)

const (
	minUnsniffedRenderableMediaBytes = 128
)

// MediaDescriptorID reports the Snapchat content object ID when the inline
// payload is not renderable media but a small protobuf descriptor that must be
// resolved through the media delivery service. Snapchat web sends these
// ~34-61 byte descriptors for chat media instead of bytes or URLs.
func MediaDescriptorID(data []byte) string {
	return ParseMediaDescriptor(data).ContentObjectID
}

func descriptorIDFromProtoBytes(value []byte) bool {
	if len(value) < minMediaDescriptorIDBytes || len(value) > maxMediaDescriptorIDBytes {
		return false
	}
	for _, b := range value {
		if b < 0x21 || b > 0x7e {
			return false
		}
	}
	return true
}

// parseProtoFields walks one protobuf message, invoking seen for every field
// until the buffer is exhausted. It rejects malformed or implausible data so
// real media bytes are never mistaken for descriptors.
func parseProtoFields(data []byte, seen func(field protowire.Number, kind protowire.Type, value []byte, varint uint64) error) error {
	for len(data) > 0 {
		number, kind, n := protowire.ConsumeTag(data)
		if n < 0 || number == 0 || number > maxMediaDescriptorFieldNum {
			return fmt.Errorf("invalid descriptor protobuf tag")
		}
		data = data[n:]
		var value []byte
		var varint uint64
		switch kind {
		case protowire.VarintType:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return fmt.Errorf("invalid descriptor protobuf varint")
			}
			varint, data = v, data[n:]
		case protowire.BytesType:
			v, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return fmt.Errorf("invalid descriptor protobuf bytes")
			}
			value, data = v, data[n:]
		default:
			return fmt.Errorf("unsupported descriptor protobuf wire type")
		}
		if err := seen(number, kind, value, varint); err != nil {
			return err
		}
	}
	return nil
}

// unpackedDescriptorVarints decodes a packed protobuf varint sequence.
func unpackedDescriptorVarints(value []byte) []uint64 {
	var result []uint64
	offset := 0
	for offset < len(value) {
		varint, next, ok := readProtoVarint(value, offset)
		if !ok {
			return nil
		}
		result = append(result, varint)
		offset = next
	}
	return result
}

// ParseMediaDescriptor recognizes the descriptor shapes actually observed on
// the wire. Raw descriptor bytes stay the source of truth for the resolver;
// the extracted ID is informational only.
func ParseMediaDescriptor(data []byte) MediaDescriptor {
	if len(data) < 2+minMediaDescriptorIDBytes || len(data) > maxMediaDescriptorBytes {
		return MediaDescriptor{}
	}
	// Shape A: field 1 holds the ASCII content object ID directly, and
	// field 2 wraps a nested ContentDescriptor (field 2 = ASCII contentId,
	// field 6 = readyLocationIds, field 9 = claimId, field 10 = use case,
	// field 12 = hostPatternVersion) for the live 61-byte capture (message
	// 745) and the 60-byte capture (message 9983). Message 745's nested
	// descriptor resolves at https://cf-st.sc-cdn.net/d/{nested contentId}
	// ?uc={field10} (verified live: HTTP 200, 491424 encrypted bytes
	// decrypted by the message's clear key/IV); message 9983 carries
	// readyLocationIds [3] and must resolve at /c/ through the network
	// mapping instead, so the static /d/ shape is only the first attempt.
	if data[0] == 0x0a {
		if length, offset, ok := readProtoVarint(data, 1); ok &&
			int(length) <= len(data)-offset &&
			descriptorIDFromProtoBytes(data[offset:offset+int(length)]) {
			rest := data[offset+int(length):]
			useCase := uint64(0)
			useCaseSeen := false
			ambiguous := false
			// Nested ContentDescriptor (field 2) contentId and use case.
			var nestedID []byte
			var nestedUseCase uint64
			var nestedUseCaseSeen bool
			var nestedReadyLocations []uint64
			nestedAmbiguous := false
			restWalker := func(field protowire.Number, kind protowire.Type, value []byte, varint uint64) error {
				if field == mediaDescriptorUseCaseField && kind == protowire.VarintType {
					if useCaseSeen && varint != useCase {
						ambiguous = true
					}
					useCase, useCaseSeen = varint, true
				}
				if field == 2 && kind == protowire.BytesType {
					var innerID []byte
					var innerUseCase uint64
					var innerUseCaseSeen bool
					var innerReadyLocations []uint64
					innerAmbiguous := false
					err := parseProtoFields(value, func(inner protowire.Number, innerKind protowire.Type, innerValue []byte, innerVarint uint64) error {
						if inner == 2 && innerKind == protowire.BytesType {
							if innerID != nil {
								innerAmbiguous = true
							}
							innerID = innerValue
						}
						if inner == mediaDescriptorUseCaseField && innerKind == protowire.VarintType {
							if innerUseCaseSeen && innerVarint != innerUseCase {
								innerAmbiguous = true
							}
							innerUseCase, innerUseCaseSeen = innerVarint, true
						}
						// readyLocationIds (repeated varint, possibly packed).
						if inner == 6 {
							switch innerKind {
							case protowire.VarintType:
								innerReadyLocations = append(innerReadyLocations, innerVarint)
							case protowire.BytesType:
								innerReadyLocations = append(innerReadyLocations, unpackedDescriptorVarints(innerValue)...)
							}
						}
						return nil
					})
					if err != nil || innerAmbiguous || innerID == nil || !descriptorIDFromProtoBytes(innerID) {
						// Not a recognizable nested ContentDescriptor; skip it.
						return nil
					}
					if nestedID != nil && !bytes.Equal(nestedID, innerID) {
						nestedAmbiguous = true
						return nil
					}
					nestedID = innerID
					if innerUseCaseSeen {
						if nestedUseCaseSeen && nestedUseCase != innerUseCase {
							nestedAmbiguous = true
							return nil
						}
						nestedUseCase, nestedUseCaseSeen = innerUseCase, true
					}
					nestedReadyLocations = append(nestedReadyLocations, innerReadyLocations...)
				}
				return nil
			}
			if len(rest) == 0 || (parseProtoFields(rest, restWalker) == nil && !ambiguous) {
				resolved := MediaDescriptor{
					Detected:         true,
					Shape:            mediaDescriptorShapeField1Direct,
					ContentObjectID:  string(data[offset : offset+int(length)]),
					UseCase:          useCase,
					ReadyLocationIDs: nestedReadyLocations,
				}
				if nestedID != nil && !nestedAmbiguous && descriptorIDFromProtoBytes(nestedID) {
					resolved.CDNObjectID = string(nestedID)
					resolved.CDNPathSegment = mediaCDNPathSegmentDirect
					resolved.CDNEligible = true
					if nestedUseCaseSeen {
						resolved.UseCase = nestedUseCase
					}
				}
				return resolved
			}
		}
	}
	// Shape B: field 2 wraps an inner protobuf whose field 2 holds the ASCII
	// content object ID. Bundle field map for the inner ContentDescriptor
	// (9846a7958a5f0bee7197.js): f2 contentId, f6 readyLocationIds (repeated
	// varint), f9 claimId, f10 useCase, f12 hostPatternVersion.
	if data[0] == 0x12 {
		if length, offset, ok := readProtoVarint(data, 1); ok && int(length) == len(data)-offset {
			inner := data[offset : offset+int(length)]
			var candidate []byte
			var f6Values [][]byte
			var f6Value []byte
			var f9, f10, f12 uint64
			var f9Seen, f10Seen, f12Seen bool
			err := parseProtoFields(inner, func(field protowire.Number, kind protowire.Type, value []byte, varint uint64) error {
				if field == 2 && kind == protowire.BytesType && descriptorIDFromProtoBytes(value) {
					if candidate != nil {
						return fmt.Errorf("ambiguous descriptor content object ID")
					}
					candidate = value
				}
				if field == 6 && kind == protowire.BytesType {
					f6Values = append(f6Values, value)
					if f6Value != nil && !bytes.Equal(f6Value, value) {
						return fmt.Errorf("ambiguous descriptor ready location ids")
					}
					if f6Value == nil {
						f6Value = value
					}
				}
				switch field {
				case 9:
					if f9Seen && f9 != varint {
						return fmt.Errorf("ambiguous descriptor field 9")
					}
					f9, f9Seen = varint, true
				case mediaDescriptorUseCaseField:
					if f10Seen && f10 != varint {
						return fmt.Errorf("ambiguous descriptor use case")
					}
					f10, f10Seen = varint, true
				case 12:
					if f12Seen && f12 != varint {
						return fmt.Errorf("ambiguous descriptor field 12")
					}
					f12, f12Seen = varint, true
				}
				return nil
			})
			if err == nil && candidate != nil {
				// readyLocationIds may be packed (one bytes value holding
				// consecutive varints) or unpacked (repeated single-varint
				// values); live captures carry one id per value.
				var readyLocations []uint64
				for _, value := range f6Values {
					readyLocations = append(readyLocations, unpackedDescriptorVarints(value)...)
				}
				return MediaDescriptor{
					Detected:         true,
					Shape:            mediaDescriptorShapeField2Nested,
					ContentObjectID:  string(candidate),
					CDNObjectID:      string(candidate),
					CDNPathSegment:   mediaCDNPathSegmentContent,
					UseCase:          f10,
					ReadyLocationIDs: readyLocations,
					// Proven static rule for location 3 only (live 4114):
					// /c/{id}?uc={f10}. Other locations resolve through the
					// bolt network mapping (media_mapping.go).
					CDNEligible: len(f6Value) == 1 && f6Value[0] == 3 && f9Seen && f9 == 2 && f12Seen && f12 == 1,
				}
			}
		}
	}
	return MediaDescriptor{}
}

// validMediaContentObjectID constrains the ID to URL-unreserved characters so
// the constructed CDN path can never smuggle path separators, query strings,
// or fragments. The length bounds mirror descriptorIDFromProtoBytes.
func validMediaContentObjectID(id string) bool {
	if len(id) < minMediaDescriptorIDBytes || len(id) > maxMediaDescriptorIDBytes {
		return false
	}
	for _, b := range []byte(id) {
		switch {
		case b >= 'A' && b <= 'Z':
		case b >= 'a' && b <= 'z':
		case b >= '0' && b <= '9':
		case b == '-' || b == '_':
		default:
			return false
		}
	}
	return true
}

// MediaContentObjectURL constructs the direct CDN retrieval URL for an
// ordinary chat-media content object, mirroring the Web bolt resolver:
// https://cf-st.sc-cdn.net/c/{id}?uc={field10} for the field2-nested shape
// and https://cf-st.sc-cdn.net/d/{nested contentId}?uc={field10} for the
// field1-direct shape. Descriptors without a proven locally-resolvable shape
// must use the resolver RPC fallback.
func MediaContentObjectURL(data []byte) (string, error) {
	descriptor := ParseMediaDescriptor(data)
	if !descriptor.Detected || !descriptor.CDNEligible || descriptor.CDNObjectID == "" {
		return "", fmt.Errorf("media CDN URL requires a locally-resolvable content descriptor")
	}
	if !validMediaContentObjectID(descriptor.CDNObjectID) {
		return "", fmt.Errorf("media content object ID is not URL-safe")
	}
	u := url.URL{
		Scheme: "https",
		Host:   mediaCDNHost,
		Path:   "/" + descriptor.CDNPathSegment + "/" + descriptor.CDNObjectID,
	}
	u.RawQuery = url.Values{mediaCDNUseCaseParam: {strconv.FormatUint(descriptor.UseCase, 10)}}.Encode()
	return u.String(), nil
}

func (c *Client) DownloadMedia(ctx context.Context, media MediaAttachment) ([]byte, string, error) {
	data, contentType, _, err := c.DownloadMediaWithInfo(ctx, media)
	return data, contentType, err
}

func (c *Client) DownloadMediaWithInfo(ctx context.Context, media MediaAttachment) ([]byte, string, MediaDownloadInfo, error) {
	info := MediaDownloadInfo{DecryptPath: "none"}
	descriptor := ParseMediaDescriptor(media.Data)
	if descriptor.Detected && strings.TrimSpace(media.URL) == "" {
		info.MediaID = descriptor.ContentObjectID
		info.DescriptorShape = descriptor.Shape
		// Proven primary path for ordinary chat media: direct CDN retrieval
		// from the content object ID (Web hB output shape). A successful CDN
		// download never touches the resolver RPC.
		if cdnURL, urlErr := MediaContentObjectURL(media.Data); urlErr == nil {
			info.Source = "descriptor-cdn"
			data, contentType, fetchErr := c.fetchMediaCDN(ctx, cdnURL)
			if fetchErr == nil {
				return c.finalizeDownloadedMedia(media, data, contentType, info)
			}
			reason := fetchErr.Error()
			if len(reason) > 300 {
				reason = reason[:300]
			}
			info.CDNFallbackReason = reason
			info.Source = "descriptor-cdn-fallback-rpc"
		}
		// General deterministic path: descriptors with readyLocationIds the
		// static /c/ and /d/ rules do not cover (e.g. message 79, location
		// 79 -> bolt-gcdn.sc-cdn.net/bq/{id}) resolve through the public
		// bolt network mapping exactly as the Web bolt resolver does.
		if mappedURL, ok, mapErr := c.mediaMappedCDNURL(ctx, media.Data); mapErr != nil {
			info.CDNFallbackReason = fmt.Sprintf("network mapping: %v", mapErr)
		} else if ok {
			info.Source = "descriptor-mapped-cdn"
			data, contentType, fetchErr := c.fetchMediaCDN(ctx, mappedURL)
			if fetchErr == nil {
				return c.finalizeDownloadedMedia(media, data, contentType, info)
			}
			reason := fetchErr.Error()
			if len(reason) > 300 {
				reason = reason[:300]
			}
			info.CDNFallbackReason = reason
		}
		// Compatibility fallback: the resolver RPC (PERMISSION_DENIED for
		// ordinary chat media in past observations, but harmless to try).
		info.Source = "descriptor-rpc"
		urls, err := c.ResolveMediaDescriptor(ctx, media.Data)
		if err != nil {
			return nil, "", info, err
		}
		for _, mediaURL := range urls {
			resolved := media
			resolved.Data = nil
			resolved.URL = mediaURL
			data, mime, downloaded, err := c.DownloadMediaWithInfo(ctx, resolved)
			if err != nil {
				continue
			}
			downloaded.Source = info.Source
			downloaded.MediaID = descriptor.ContentObjectID
			downloaded.DescriptorShape = descriptor.Shape
			return data, mime, downloaded, nil
		}
		return nil, "", info, fmt.Errorf("all resolved Snapchat media downloads failed validation or retrieval")
	}
	var data []byte
	contentType := strings.TrimSpace(media.MimeType)
	if len(media.Data) > 0 && !descriptor.Detected {
		info.Source = "inline"
		data = append([]byte(nil), media.Data...)
	} else if strings.TrimSpace(media.URL) != "" {
		info.Source = "url"
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, media.URL, nil)
		if err != nil {
			return nil, "", info, err
		}
		if c.userAgent != "" {
			req.Header.Set("User-Agent", c.userAgent)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, "", info, err
		}
		defer resp.Body.Close()
		if resp.StatusCode >= 300 {
			return nil, "", info, fmt.Errorf("download Snapchat media returned %s", resp.Status)
		}
		if contentType == "" {
			contentType = strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
		}
		data, err = io.ReadAll(io.LimitReader(resp.Body, 64*1024*1024+1))
		if err != nil {
			return nil, "", info, err
		}
		if len(data) > 64*1024*1024 {
			return nil, "", info, fmt.Errorf("Snapchat media exceeds 64 MiB download limit")
		}
	} else {
		return nil, "", info, fmt.Errorf("Snapchat media has no URL or inline data")
	}
	return c.finalizeDownloadedMedia(media, data, contentType, info)
}

// fetchMediaCDN performs the CDN GET exactly as the Web page does: a plain
// cross-origin fetch with no credentials and no auth-header injection
// (17196.u9 = fetch(request, {signal})). Only normal browser request metadata
// is sent: user agent, accept, origin, referer, and sec-fetch hints. Snapchat
// cookies, the bearer token, and x-snap-* / mcs-* API headers must never be
// sent to the CDN.
func (c *Client) fetchMediaCDN(ctx context.Context, cdnURL string) ([]byte, string, error) {
	if c == nil || c.http == nil {
		return nil, "", fmt.Errorf("snapchat CDN fetch has no authenticated client")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cdnURL, nil)
	if err != nil {
		return nil, "", err
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://www.snapchat.com")
	req.Header.Set("Referer", "https://www.snapchat.com/")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "cross-site")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("Snapchat CDN media returned %s", resp.Status)
	}
	contentType := strings.TrimSpace(strings.Split(resp.Header.Get("Content-Type"), ";")[0])
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024*1024+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > 64*1024*1024 {
		return nil, "", fmt.Errorf("Snapchat media exceeds 64 MiB download limit")
	}
	return data, contentType, nil
}

// finalizeDownloadedMedia applies the shared post-download pipeline: decrypt
// when clear-key metadata is present, then validate and MIME-sniff the clear
// bytes. Descriptor payloads can never pass validation, so a descriptor can
// never be returned as rendered media.
func (c *Client) finalizeDownloadedMedia(media MediaAttachment, data []byte, contentType string, info MediaDownloadInfo) ([]byte, string, MediaDownloadInfo, error) {
	if len(data) == 0 {
		return nil, "", info, fmt.Errorf("Snapchat media was empty")
	}
	if len(media.Key) > 0 || len(media.IV) > 0 {
		info.CiphertextBytes = len(data)
		decrypted, ok := decryptMediaCBC(data, media.Key, media.IV)
		if !ok {
			return nil, "", info, fmt.Errorf("Snapchat media clear-key decryption failed")
		}
		data = decrypted
		info.DecryptPath = "cbc-clear-media-key"
	}
	if err := ValidateRenderableMediaPayload(data, contentType, media.Kind); err != nil {
		return nil, "", info, err
	}
	if detected := mediaMagicMime(data); detected != "" {
		contentType = detected
	} else if contentType == "" || contentType == "application/octet-stream" {
		contentType = detectMediaMime(data, media.Kind)
	}
	info.Bytes = len(data)
	info.ContentType = contentType
	return data, contentType, info, nil
}

func ValidateRenderableMediaPayload(data []byte, mimeType string, kind MediaKind) error {
	if len(data) == 0 {
		return fmt.Errorf("Snapchat media was empty")
	}
	if id := MediaDescriptorID(data); id != "" {
		return fmt.Errorf("snapchat media is a descriptor reference (media_id=%s) with no fetchable URL; downloading it requires the media service integration", id)
	}
	detected := mediaMagicMime(data)
	if detected == "image/jpeg" || detected == "image/png" || detected == "image/gif" {
		config, _, err := image.DecodeConfig(bytes.NewReader(data))
		if err != nil || config.Width <= 0 || config.Height <= 0 {
			return fmt.Errorf("invalid Snapchat image structure: %v", err)
		}
		if int64(config.Width)*int64(config.Height) > 40_000_000 {
			return fmt.Errorf("Snapchat image exceeds 40 megapixel validation limit")
		}
		if _, _, err := image.Decode(bytes.NewReader(data)); err != nil {
			return fmt.Errorf("invalid Snapchat image data: %w", err)
		}
		return nil
	}
	if detected == "video/mp4" {
		return validateMP4(data)
	}
	mimeType = strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	if len(data) < minUnsniffedRenderableMediaBytes && (kind == MediaKindImage || kind == MediaKindVideo || kind == MediaKindGIF || strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/")) {
		return fmt.Errorf("snapchat media payload is too small and lacks image/video magic bytes (bytes=%d mime=%q kind=%s)", len(data), mimeType, kind)
	}
	if kind == MediaKindImage || kind == MediaKindGIF || kind == MediaKindVideo || strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/") {
		return fmt.Errorf("Snapchat media lacks a supported, valid image/video structure (bytes=%d)", len(data))
	}
	return nil
}

// Reject header-only and truncated ISO BMFF payloads. Track metadata and actual
// media data independently: an ftyp signature alone is not a playable video.
func validateMP4(data []byte) error {
	var metadata, payload bool
	for offset := 0; offset < len(data); {
		if len(data)-offset < 8 {
			return fmt.Errorf("truncated MP4 box header")
		}
		size := uint64(binary.BigEndian.Uint32(data[offset:]))
		kind := string(data[offset+4 : offset+8])
		header := uint64(8)
		if size == 1 {
			if len(data)-offset < 16 {
				return fmt.Errorf("truncated MP4 extended box")
			}
			size = binary.BigEndian.Uint64(data[offset+8:])
			header = 16
		} else if size == 0 {
			size = uint64(len(data) - offset)
		}
		if size < header || size > uint64(len(data)-offset) {
			return fmt.Errorf("invalid MP4 box length")
		}
		metadata = metadata || (kind == "moov" && size > header)
		payload = payload || (kind == "mdat" && size > header)
		offset += int(size)
	}
	if !metadata || !payload {
		return fmt.Errorf("MP4 missing movie metadata or media payload")
	}
	return nil
}

func mediaAttachmentsFromEnvelope(envelope *protos.ContentEnvelope, messageID string) []MediaAttachment {
	if envelope == nil {
		return nil
	}
	key, iv := mediaKeyFromEnvelope(envelope)
	// References the envelope itself declares as thumbnails are server-derived
	// renditions of the primary media (e.g. message 772's mediaListId=2
	// ".1020" variant). They are not user-visible content: Matrix clients
	// derive their own thumbnails, and the rendition object is not retrievable
	// at the deterministic CDN path. Excluding them keeps the primary media as
	// the message's single bridgeable attachment.
	thumbnailListIDs := make(map[uint64]bool)
	for _, thumb := range envelope.GetThumbnails().GetThumbnails() {
		if mediaID := thumb.GetMediaId(); mediaID != nil {
			thumbnailListIDs[mediaID.GetMediaListId()] = true
		}
	}
	media := make([]MediaAttachment, 0)
	for index, info := range envelope.GetRemoteMediaInfos() {
		kind, mimeType := mediaKindFromRemote(info.GetMediaType(), info.GetHasAudio())
		attachment := MediaAttachment{
			ID:       stableMediaID(messageID, "remote", fmt.Sprint(index), info.GetContentUrl(), info.GetLegacyMediaId()),
			URL:      strings.TrimSpace(info.GetContentUrl()),
			FileName: mediaFileName(messageID, index, kind, mimeType),
			MimeType: mimeType,
			Kind:     kind,
			Data:     append([]byte(nil), info.GetContentObject()...),
			Key:      key,
			IV:       iv,
		}
		if attachment.URL != "" || len(attachment.Data) > 0 {
			media = append(media, attachment)
		}
	}
	for listIndex, list := range envelope.GetMediaReferenceLists() {
		for refIndex, ref := range list.GetReference() {
			if thumbnailListIDs[ref.GetMediaListId()] {
				continue
			}
			kind, mimeType := mediaKindFromReference(ref.GetMediaType())
			attachment := MediaAttachment{
				ID:       stableMediaID(messageID, "ref", fmt.Sprint(listIndex), fmt.Sprint(refIndex), ref.GetUrl(), fmt.Sprint(ref.GetMediaListId())),
				URL:      strings.TrimSpace(ref.GetUrl()),
				FileName: mediaFileName(messageID, len(media), kind, mimeType),
				MimeType: mimeType,
				Kind:     kind,
				Data:     append([]byte(nil), ref.GetContentObject()...),
				Key:      key,
				IV:       iv,
			}
			if attachment.URL != "" || len(attachment.Data) > 0 {
				media = append(media, attachment)
			}
		}
	}
	return media
}

func mediaKeyFromEnvelope(envelope *protos.ContentEnvelope) ([]byte, []byte) {
	if envelope == nil {
		return nil, nil
	}
	clear := envelope.GetEnvelopeEncryption().GetClearTextMediaKey()
	if clear == nil {
		return nil, nil
	}
	return append([]byte(nil), clear.GetMediaKey()...), append([]byte(nil), clear.GetMediaIv()...)
}

func mediaKindFromRemote(raw int32, hasAudio bool) (MediaKind, string) {
	switch protos.ContentEnvelope_RemoteMediaInfo_MediaType(raw) {
	case protos.ContentEnvelope_RemoteMediaInfo_MediaType_IMAGE:
		return MediaKindImage, "image/jpeg"
	case protos.ContentEnvelope_RemoteMediaInfo_MediaType_VIDEO:
		return MediaKindVideo, "video/mp4"
	case protos.ContentEnvelope_RemoteMediaInfo_MediaType_GIF:
		return MediaKindGIF, "image/gif"
	default:
		if hasAudio {
			return MediaKindVideo, "video/mp4"
		}
		return MediaKindFile, "application/octet-stream"
	}
}

func mediaKindFromReference(raw protos.MediaType) (MediaKind, string) {
	switch raw {
	case protos.MediaType_MEDIA_TYPE_IMAGE:
		return MediaKindImage, "image/jpeg"
	case protos.MediaType_MEDIA_TYPE_VIDEO, protos.MediaType_MEDIA_TYPE_VIDEONOAUDIO:
		return MediaKindVideo, "video/mp4"
	case protos.MediaType_MEDIA_TYPE_ANIMATEDIMAGE:
		return MediaKindGIF, "image/gif"
	case protos.MediaType_MEDIA_TYPE_AUDIO:
		return MediaKindFile, "audio/mpeg"
	default:
		return MediaKindFile, "application/octet-stream"
	}
}

func mediaFileName(messageID string, index int, kind MediaKind, mimeType string) string {
	ext := ".bin"
	switch {
	case strings.Contains(mimeType, "jpeg"):
		ext = ".jpg"
	case strings.Contains(mimeType, "png"):
		ext = ".png"
	case strings.Contains(mimeType, "gif"):
		ext = ".gif"
	case strings.Contains(mimeType, "mp4"):
		ext = ".mp4"
	case strings.Contains(mimeType, "mpeg"):
		ext = ".mp3"
	case kind == MediaKindImage:
		ext = ".jpg"
	case kind == MediaKindVideo:
		ext = ".mp4"
	}
	return fmt.Sprintf("snap-%s-%d%s", sanitizeFileNamePart(messageID), index+1, ext)
}

func sanitizeFileNamePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "media"
	}
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	if builder.Len() == 0 {
		return "media"
	}
	return builder.String()
}

func stableMediaID(parts ...string) string {
	hash := sha1.Sum([]byte(strings.Join(parts, "::")))
	return hex.EncodeToString(hash[:])[:16]
}

func detectMediaMime(data []byte, kind MediaKind) string {
	if mimeType := mediaMagicMime(data); mimeType != "" {
		return mimeType
	}
	detected := http.DetectContentType(data)
	if detected != "application/octet-stream" {
		return detected
	}
	switch kind {
	case MediaKindImage:
		return "image/jpeg"
	case MediaKindVideo:
		return "video/mp4"
	case MediaKindGIF:
		return "image/gif"
	default:
		return detected
	}
}

func mediaMagicMime(data []byte) string {
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return "image/jpeg"
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png"
	case len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a"):
		return "image/gif"
	case len(data) >= 16 && string(data[4:8]) == "ftyp":
		return "video/mp4"
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WAVE":
		return "audio/wav"
	case len(data) >= 3 && string(data[:3]) == "ID3":
		return "audio/mpeg"
	case len(data) >= 2 && data[0] == 0xff && (data[1] == 0xfb || data[1] == 0xf3 || data[1] == 0xf2):
		return "audio/mpeg"
	default:
		return ""
	}
}

func decryptMediaCBC(data, key, iv []byte) ([]byte, bool) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, false
	}
	if len(iv) != aes.BlockSize || len(data) == 0 || len(data)%aes.BlockSize != 0 {
		return nil, false
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, false
	}
	output := make([]byte, len(data))
	cipher.NewCBCDecrypter(block, iv[:aes.BlockSize]).CryptBlocks(output, data)
	output, ok := pkcs7Unpad(output, aes.BlockSize)
	if !ok {
		return nil, false
	}
	return output, true
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, bool) {
	if len(data) == 0 || len(data)%blockSize != 0 {
		return nil, false
	}
	pad := int(data[len(data)-1])
	if pad == 0 || pad > blockSize || pad > len(data) {
		return nil, false
	}
	for _, value := range data[len(data)-pad:] {
		if int(value) != pad {
			return nil, false
		}
	}
	return data[:len(data)-pad], true
}

func previewFromDisplayInfo(info *protos.DisplayInfo) (string, bool) {
	if info == nil {
		return "", false
	}
	if snap := info.GetSnapItem(); snap != nil {
		if snap.GetState() == protos.SnapItemState_VIEWED {
			return "Opened Snap", false
		}
		if snap.GetHasAudio() {
			return "New Snap with audio", true
		}
		return "New Snap", snap.GetState() != protos.SnapItemState_VIEWED
	}
	if chat := info.GetChatItem(); chat != nil {
		switch chat.GetState() {
		case protos.ChatItemState_CHAT_UNVIEWED, protos.ChatItemState_CHAT_SAVED_UNVIEWED,
			protos.ChatItemState_CHAT_SCREENSHOTTED_UNVIEWED, protos.ChatItemState_CHAT_RECORDED_UNVIEWED:
			return "New message", true
		case protos.ChatItemState_CHAT_VIEWED, protos.ChatItemState_CHAT_SAVED_VIEWED:
			return "Chat", false
		default:
			return chat.GetState().String(), false
		}
	}
	if call := info.GetCallItem(); call != nil {
		if call.GetIsVideo() {
			return "Missed video call", true
		}
		return "Missed call", true
	}
	if conv := info.GetConversationItem(); conv != nil {
		return conv.GetState().String(), false
	}
	return "", false
}
