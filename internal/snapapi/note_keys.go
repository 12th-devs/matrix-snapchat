package snapapi

import (
	"encoding/base64"
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// NoteAudioEncryptionKeys extracts the voice-note AES key/IV and duration from
// decoded NOTE contents: Contents.note(6) -> Note.note(1, audio oneof) ->
// AudioNote.note(1) = AudioNoteDetails{encryptionInfo(4){key(1) b64,
// iv(2) b64}, mediaDurationMs(13)}. This is the receive-side counterpart of
// the note key/IV injection the web sender performs (updateMessageContent).
func NoteAudioEncryptionKeys(contents []byte) (key, iv []byte, durationMs int64, err error) {
	if len(contents) == 0 {
		return nil, nil, 0, fmt.Errorf("empty note contents")
	}
	notes, err := mediaProtoField(contents, 6)
	if err != nil || len(notes) != 1 {
		return nil, nil, 0, fmt.Errorf("note contents missing (field 6): %w", err)
	}
	audioOneof, err := mediaProtoField(notes[0], 1)
	if err != nil || len(audioOneof) != 1 {
		return nil, nil, 0, fmt.Errorf("note audio oneof missing (6.1): %w", err)
	}
	audioNote, err := mediaProtoField(audioOneof[0], 1)
	if err != nil || len(audioNote) != 1 {
		return nil, nil, 0, fmt.Errorf("note audio details missing (6.1.1): %w", err)
	}
	details := audioNote[0]
	enc, err := mediaProtoField(details, 4)
	if err != nil || len(enc) != 1 {
		return nil, nil, 0, fmt.Errorf("note encryptionInfo missing (6.1.1.4): %w", err)
	}
	keyB64, err := mediaProtoField(enc[0], 1)
	if err != nil || len(keyB64) != 1 {
		return nil, nil, 0, fmt.Errorf("note key missing (6.1.1.4.1): %w", err)
	}
	ivB64, err := mediaProtoField(enc[0], 2)
	if err != nil || len(ivB64) != 1 {
		return nil, nil, 0, fmt.Errorf("note iv missing (6.1.1.4.2): %w", err)
	}
	key, err = base64.StdEncoding.DecodeString(string(keyB64[0]))
	if err != nil {
		return nil, nil, 0, fmt.Errorf("note key not base64: %w", err)
	}
	iv, err = base64.StdEncoding.DecodeString(string(ivB64[0]))
	if err != nil {
		return nil, nil, 0, fmt.Errorf("note iv not base64: %w", err)
	}
	offset := 0
	for offset < len(details) {
		tag, kind, n := protowire.ConsumeTag(details[offset:])
		if n < 0 {
			break
		}
		offset += n
		field := tag
		if field == 13 {
			value, n2 := protowire.ConsumeVarint(details[offset:])
			if n2 >= 0 {
				durationMs = int64(value)
			}
			break
		}
		n2 := protowire.ConsumeFieldValue(field, kind, details[offset:])
		if n2 < 0 {
			break
		}
		offset += n2
	}
	return key, iv, durationMs, nil
}
