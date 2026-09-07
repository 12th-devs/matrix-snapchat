package snapapi

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/0xzer/snapper/data/paths"
	"github.com/0xzer/snapper/protos"
	"google.golang.org/protobuf/encoding/protowire"
)

// Read-only message structure inspector for operator diagnostics. It fetches a
// small message window for ONE conversation, locates ONE message, and reports
// the decoded envelope/contents structure without changing any production
// behavior. Large blobs are truncated; strings are preserved for inspection.

const (
	inspectMaxDepth        = 14
	inspectMaxTextLength   = 240
	inspectMaxHexPrefix    = 48
	inspectDefaultFetchLim = 10
)

type InspectNode struct {
	Field    int            `json:"field,omitempty"`
	Wire     int            `json:"wire,omitempty"`
	Varint   uint64         `json:"varint,omitempty"`
	Text     string         `json:"text,omitempty"`
	Len      int            `json:"len,omitempty"`
	Hex      string         `json:"hex,omitempty"`
	Children []*InspectNode `json:"children,omitempty"`
}

type InspectEncryption struct {
	Method        string `json:"method"`
	MediaKeyLen   int    `json:"mediaKeyLen,omitempty"`
	MediaIVLen    int    `json:"mediaIvLen,omitempty"`
	CEKLen        int    `json:"cekLen,omitempty"`
	CEKIVLen      int    `json:"cekIvLen,omitempty"`
	NonceLen      int    `json:"nonceLen,omitempty"`
	SenderKeyLen  int    `json:"senderPublicKeyLen,omitempty"`
	SenderVersion int32  `json:"senderVersion,omitempty"`
}

// EELDecryptDiagnostics mirrors the connector's decrypt-attribution fields so
// reports can prove the plaintext belongs to the requested message.
type EELDecryptDiagnostics struct {
	Method              string `json:"method,omitempty"`
	RequestedMessageID  string `json:"requestedMessageId,omitempty"`
	MatchedWebMessageID string `json:"matchedWebMessageId,omitempty"`
	ContentSource       string `json:"contentSource,omitempty"`
	MessageCount        int    `json:"messageCount,omitempty"`
	ExactMatch          *bool  `json:"exactMatch,omitempty"`
}

// EELDiagnosticsProvider is optionally implemented by EELDecrypter adapters
// that can expose diagnostics from their most recent decrypt call.
type EELDiagnosticsProvider interface {
	LastEELDecryptDiagnostics() *EELDecryptDiagnostics
}

type InspectMediaRef struct {
	Index            int    `json:"index"`
	ListIndex        int    `json:"listIndex"`
	Role             string `json:"role"`
	Oneof            string `json:"oneof,omitempty"`
	ContentObjectID  string `json:"contentObjectId,omitempty"`
	DescriptorShape  string `json:"descriptorShape,omitempty"`
	ContentObjectID2 string `json:"descriptorContentId,omitempty"`
	Len              int    `json:"bytes"`
	Hex              string `json:"hex,omitempty"`
	URL              string `json:"url,omitempty"`
	MediaType        string `json:"mediaType,omitempty"`
	MediaListID      uint64 `json:"mediaListId,omitempty"`
}

type MessageInspectReport struct {
	ChatID              string                 `json:"chatId"`
	MessageID           string                 `json:"messageId"`
	SenderID            string                 `json:"senderId,omitempty"`
	ClientResolutionID  uint64                 `json:"clientResolutionId,omitempty"`
	ContentType         string                 `json:"contentType"`
	SavePolicy          string                 `json:"savePolicy,omitempty"`
	Encryption          *InspectEncryption     `json:"encryption,omitempty"`
	RemoteMediaInfos    []*InspectMediaRef     `json:"remoteMediaInfos,omitempty"`
	MediaReferenceLists []*InspectMediaRef     `json:"mediaReferenceLists,omitempty"`
	ContentsLen         int                    `json:"contentsLen"`
	ContentsSource      string                 `json:"contentsSource"`
	ContentsSha256      string                 `json:"contentsSha256,omitempty"`
	EEL                 *EELDecryptDiagnostics `json:"eel,omitempty"`
	// EEL wire fields (base64) let operator tools replay /session/eel-decrypt
	// directly when the production decode path failed. Read-only diagnostics.
	EELContentB64    string `json:"eelContentB64,omitempty"`
	EELCEKB64        string `json:"eelCekB64,omitempty"`
	EELCEKIVB64      string `json:"eelCekIvB64,omitempty"`
	EELNonceB64      string `json:"eelNonceB64,omitempty"`
	EELSenderPubB64  string `json:"eelSenderPublicKeyB64,omitempty"`
	EELSenderVersion int32  `json:"eelSenderVersion,omitempty"`
	ContentsTree        []*InspectNode         `json:"contentsTree,omitempty"`
	TextFields          []string               `json:"textFields,omitempty"`
	SearchTerm          string                 `json:"searchTerm,omitempty"`
	SearchHits          []string               `json:"searchHits,omitempty"`
}

// InspectMessage fetches a small window for one conversation, finds messageID,
// and returns its decoded structure. The contents payload is decoded through
// the production envelopeContentsForDecode path (including EEL via the
// configured decrypter).
func (c *Client) InspectMessage(ctx context.Context, chatID, messageID string, fetchLimit int, searchTerm string) (*MessageInspectReport, error) {
	if strings.TrimSpace(chatID) == "" || strings.TrimSpace(messageID) == "" {
		return nil, fmt.Errorf("chat id and message id are required")
	}
	if fetchLimit <= 0 {
		fetchLimit = inspectDefaultFetchLim
	}
	convID, err := encodeUUIDString(chatID)
	if err != nil {
		return nil, err
	}
	var version int64
	if conv, err := c.conversation(ctx, chatID); err == nil && conv != nil {
		version = conv.GetVersion()
	}
	messages := c.inspectQueryMessages(ctx, convID, version, fetchLimit)
	if len(messages) == 0 && version != 0 {
		messages = c.inspectQueryMessages(ctx, convID, 0, fetchLimit)
	}
	var target *protos.ContentMessage
	for _, msg := range messages {
		id := fmt.Sprintf("%d", msg.GetMessageId())
		if id == messageID || fmt.Sprintf("client-%d", msg.GetClientResolutionId()) == messageID {
			target = msg
			break
		}
	}
	if target == nil {
		found := make([]string, 0, len(messages))
		for _, msg := range messages {
			found = append(found, fmt.Sprintf("%d", msg.GetMessageId()))
		}
		return nil, fmt.Errorf("message %s not found in the %d-message window (ids: %s)", messageID, len(messages), strings.Join(found, ", "))
	}
	return c.buildInspectReport(ctx, chatID, messageID, target, searchTerm), nil
}

func (c *Client) inspectQueryMessages(ctx context.Context, convID *protos.UUID, version int64, limit int) []*protos.ContentMessage {
	req := &protos.QueryMessagesRequest{
		SelfUserId:         c.selfUUID(),
		ConversationId:     convID,
		RequestedCountSize: int32(limit),
		CurrentVersion:     version,
	}
	var resp protos.QueryMessagesResponse
	if err := c.doGRPC(ctx, paths.QUERY_MESSAGES, req, &resp); err != nil {
		return nil
	}
	return resp.GetMessages()
}

func (c *Client) buildInspectReport(ctx context.Context, chatID, messageID string, msg *protos.ContentMessage, searchTerm string) *MessageInspectReport {
	envelope := msg.GetContents()
	report := &MessageInspectReport{
		ChatID:             chatID,
		MessageID:          fmt.Sprintf("%d", msg.GetMessageId()),
		SenderID:           uuidToString(msg.GetSenderId()),
		ClientResolutionID: msg.GetClientResolutionId(),
		ContentType:        envelope.GetContentType().String(),
		SavePolicy:         envelope.GetSavePolicy().String(),
	}
	report.Encryption = inspectEncryption(envelope)
	for index, info := range envelope.GetRemoteMediaInfos() {
		report.RemoteMediaInfos = append(report.RemoteMediaInfos, inspectRemoteMediaInfo(index, info))
	}
	for listIndex, list := range envelope.GetMediaReferenceLists() {
		for refIndex, ref := range list.GetReference() {
			report.MediaReferenceLists = append(report.MediaReferenceLists, inspectMediaReference(listIndex, refIndex, ref))
		}
	}
	contents := c.envelopeContentsForDecode(ctx, chatID, messageID, envelope)
	report.ContentsLen = len(contents)
	report.ContentsSource = c.contentsSourceName(envelope, contents)
	if eel := envelope.GetEnvelopeEncryption().GetEelEncryption(); eel != nil {
		// Always populated from the wire so a failed production decode can be
		// replayed manually with the connector's decrypt endpoint.
		if raw := envelope.GetContents(); len(raw) > 0 {
			report.EELContentB64 = base64.StdEncoding.EncodeToString(raw)
		}
		report.EELCEKB64 = base64.StdEncoding.EncodeToString(eel.GetCek())
		report.EELCEKIVB64 = base64.StdEncoding.EncodeToString(eel.GetCekIv())
		report.EELNonceB64 = base64.StdEncoding.EncodeToString(eel.GetNonce())
		report.EELSenderPubB64 = base64.StdEncoding.EncodeToString(eel.GetSenderPublicKey())
		report.EELSenderVersion = eel.GetSenderVersion()
	}
	if len(contents) > 0 {
		hash := sha256.Sum256(contents)
		report.ContentsSha256 = hex.EncodeToString(hash[:])
	}
	if provider, ok := c.eelDecrypter.(EELDiagnosticsProvider); ok {
		if diagnostics := provider.LastEELDecryptDiagnostics(); diagnostics != nil {
			report.EEL = diagnostics
		}
	}
	if len(contents) > 0 {
		report.ContentsTree = walkInspectProto(contents, 0)
		collectTextFields(report.ContentsTree, &report.TextFields, "")
		if strings.TrimSpace(searchTerm) != "" {
			report.SearchTerm = searchTerm
			report.SearchHits = searchInspectTree(report.ContentsTree, searchTerm, "")
		}
	}
	return report
}

func (c *Client) contentsSourceName(envelope *protos.ContentEnvelope, contents []byte) string {
	if len(contents) == 0 {
		return "unavailable"
	}
	if clear := envelope.GetEnvelopeEncryption().GetClearTextEelKeyEncryption(); clear != nil && len(contents) > 0 {
		if _, ok := decryptEELGCM(envelope.GetContents(), clear.GetCek(), clear.GetCekIv()); ok {
			return "eel-decrypted(clearTextEelKey)"
		}
	}
	if eel := envelope.GetEnvelopeEncryption().GetEelEncryption(); eel != nil && len(envelope.GetContents()) != len(contents) {
		return "eel-decrypted"
	}
	if len(envelope.GetContents()) == len(contents) {
		return "plain"
	}
	return "decoded"
}

func inspectEncryption(envelope *protos.ContentEnvelope) *InspectEncryption {
	if envelope == nil {
		return nil
	}
	enc := envelope.GetEnvelopeEncryption()
	if enc == nil {
		return nil
	}
	out := &InspectEncryption{Method: fmt.Sprintf("%T", enc.GetMethod())}
	if clear := enc.GetClearTextMediaKey(); clear != nil {
		out.MediaKeyLen = len(clear.GetMediaKey())
		out.MediaIVLen = len(clear.GetMediaIv())
	}
	if clear := enc.GetClearTextEelKeyEncryption(); clear != nil {
		out.CEKLen = len(clear.GetCek())
		out.CEKIVLen = len(clear.GetCekIv())
	}
	if eel := enc.GetEelEncryption(); eel != nil {
		out.CEKLen = len(eel.GetCek())
		out.CEKIVLen = len(eel.GetCekIv())
		out.NonceLen = len(eel.GetNonce())
		out.SenderKeyLen = len(eel.GetSenderPublicKey())
		out.SenderVersion = eel.GetSenderVersion()
	}
	return out
}

func inspectRemoteMediaInfo(index int, info *protos.ContentEnvelope_RemoteMediaInfo) *InspectMediaRef {
	ref := &InspectMediaRef{
		Index:     index,
		Role:      "remoteMediaInfo",
		MediaType: protos.ContentEnvelope_RemoteMediaInfo_MediaType(info.GetMediaType()).String(),
	}
	switch media := info.GetMediaInfo().(type) {
	case *protos.ContentEnvelope_RemoteMediaInfo_ContentObject:
		ref.Oneof = "contentObject"
		ref.Len = len(media.ContentObject)
		ref.Hex = hexPrefix(media.ContentObject)
		ref.MediaListID = 0
	case *protos.ContentEnvelope_RemoteMediaInfo_LegacyMediaId:
		ref.Oneof = "legacyMediaId"
		ref.ContentObjectID = media.LegacyMediaId
	case *protos.ContentEnvelope_RemoteMediaInfo_ContentUrl:
		ref.Oneof = "contentUrl"
		ref.URL = media.ContentUrl
	default:
		ref.Oneof = fmt.Sprintf("%T", info.GetMediaInfo())
	}
	if descriptor := ParseMediaDescriptor(info.GetContentObject()); descriptor.Detected {
		ref.DescriptorShape = descriptor.Shape
		ref.ContentObjectID = descriptor.ContentObjectID
		ref.ContentObjectID2 = descriptor.CDNObjectID
	}
	ref.HasAudioNote(info.GetHasAudio())
	return ref
}

// HasAudioNote records the has-audio flag without changing the JSON shape.
func (r *InspectMediaRef) HasAudioNote(hasAudio bool) {
	if hasAudio {
		r.MediaType += "+audio"
	}
}

func inspectMediaReference(listIndex, refIndex int, ref *protos.MediaReference) *InspectMediaRef {
	out := &InspectMediaRef{
		Index:       refIndex,
		ListIndex:   listIndex,
		Role:        "mediaReference",
		Len:         len(ref.GetContentObject()),
		Hex:         hexPrefix(ref.GetContentObject()),
		URL:         ref.GetUrl(),
		MediaType:   ref.GetMediaType().String(),
		MediaListID: ref.GetMediaListId(),
	}
	if descriptor := ParseMediaDescriptor(ref.GetContentObject()); descriptor.Detected {
		out.DescriptorShape = descriptor.Shape
		out.ContentObjectID = descriptor.ContentObjectID
		out.ContentObjectID2 = descriptor.CDNObjectID
	}
	return out
}

// walkInspectProto converts a protobuf message into a display tree. Fields
// keep their numbers and wire types; length-delimited values become text
// (printable), nested messages (clean parse), or truncated blobs.
func walkInspectProto(data []byte, depth int) []*InspectNode {
	if depth > inspectMaxDepth {
		return nil
	}
	var nodes []*InspectNode
	for len(data) > 0 {
		number, kind, n := protowire.ConsumeTag(data)
		if n < 0 {
			nodes = append(nodes, &InspectNode{Text: "<malformed tag>"})
			return nodes
		}
		data = data[n:]
		node := &InspectNode{Field: int(number), Wire: int(kind)}
		switch kind {
		case protowire.VarintType:
			value, n := protowire.ConsumeVarint(data)
			if n < 0 {
				node.Text = "<malformed varint>"
				nodes = append(nodes, node)
				return nodes
			}
			node.Varint = value
			data = data[n:]
		case protowire.BytesType:
			value, n := protowire.ConsumeBytes(data)
			if n < 0 {
				node.Text = "<malformed bytes>"
				nodes = append(nodes, node)
				return nodes
			}
			data = data[n:]
			node.Len = len(value)
			inspectFillBytesNode(node, value, depth)
		case protowire.Fixed32Type:
			if len(data) < 4 {
				node.Text = "<truncated fixed32>"
				data = nil
			} else {
				node.Hex = hexPrefix(data[:4])
				data = data[4:]
			}
		case protowire.Fixed64Type:
			if len(data) < 8 {
				node.Text = "<truncated fixed64>"
				data = nil
			} else {
				node.Hex = hexPrefix(data[:8])
				data = data[8:]
			}
		default:
			node.Text = "<unsupported wire type>"
			nodes = append(nodes, node)
			return nodes
		}
		nodes = append(nodes, node)
	}
	return nodes
}

func inspectFillBytesNode(node *InspectNode, value []byte, depth int) {
	if len(value) == 0 {
		node.Text = "<empty>"
		return
	}
	if printableInspectText(value) {
		node.Text = truncateInspectText(string(value))
		return
	}
	if children := walkInspectProto(value, depth+1); len(children) > 0 {
		node.Children = children
		return
	}
	node.Hex = hexPrefix(value)
}

func printableInspectText(value []byte) bool {
	if !utf8.Valid(value) {
		return false
	}
	for _, b := range value {
		if b < 0x20 && b != '\n' && b != '\r' && b != '\t' {
			return false
		}
		if b == 0x7f {
			return false
		}
	}
	return true
}

func truncateInspectText(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > inspectMaxTextLength {
		return value[:inspectMaxTextLength] + "…"
	}
	return value
}

func hexPrefix(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	prefix := value
	if len(prefix) > inspectMaxHexPrefix {
		prefix = prefix[:inspectMaxHexPrefix]
	}
	out := hex.EncodeToString(prefix)
	if len(value) > inspectMaxHexPrefix {
		out += "…"
	}
	return out
}

func collectTextFields(nodes []*InspectNode, out *[]string, path string) {
	for _, node := range nodes {
		nodePath := fmt.Sprintf("%s/%d", path, node.Field)
		if node.Text != "" && node.Text != "<empty>" && !strings.HasPrefix(node.Text, "<") {
			*out = append(*out, nodePath+" = "+node.Text)
		}
		if len(node.Children) > 0 {
			collectTextFields(node.Children, out, nodePath)
		}
	}
}

func searchInspectTree(nodes []*InspectNode, needle, path string) []string {
	var hits []string
	for _, node := range nodes {
		nodePath := fmt.Sprintf("%s/%d", path, node.Field)
		if node.Text != "" && strings.Contains(node.Text, needle) {
			hits = append(hits, nodePath+" = "+node.Text)
		}
		if len(node.Children) > 0 {
			hits = append(hits, searchInspectTree(node.Children, needle, nodePath)...)
		}
	}
	return hits
}

// SortStrings is a tiny helper so reports are stable for diffing.
func SortStrings(values []string) {
	sort.Strings(values)
}
