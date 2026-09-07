package snapapi

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"github.com/0xzer/snapper/protos"
)

// Outgoing voice-note support (EXPERIMENTAL, audio/mp4 first format).
//
// The NOTE/AUDIO wire shape below is proven two ways:
//  1. Snapchat Web voice-note sender (bundle modules 56639 sendVoiceNote,
//     76748 uploadMedia/updateMessageContent, 15491 Note/AudioNote protobuf
//     encoders, 64069 AudioNoteDetails encoder).
//  2. A real incoming voice note (NOTE content type, LIFETIME save policy)
//     whose decoded contents match this structure field-for-field.
//
// The audio CONTAINER is not yet proven: Snapchat Web's voice-note recorder
// was not reachable for inspection. The web bundle contains an in-house MP4
// muxer that writes AAC audio tracks, so audio/mp4 is implemented as the
// experimental first format; it is NOT confirmed as Snapchat's canonical
// voice-note container. Other containers (audio/ogg, audio/webm, audio/mpeg)
// are rejected cleanly rather than transcoded.
//
// Content structure (Contents.proto):
//
//	Contents.note(6) -> Note.note(1, oneof audio) -> AudioNote
//	AudioNote.note(1) -> AudioNoteDetails, AudioNote.userLocale(3)
//	AudioNoteDetails.type(2)=4 (AUDIO, web hW enum), .encryptionInfo(4)
//	{key(1) b64, iv(2) b64}, .hasSound(8), .duration(12)
//	{displayDuration.durationSeconds(3)}, .mediaDurationMs(13)
//
// There is no separate media metadata wrapper, no playbackCharacteristics,
// no dimensions, and no waveform anywhere in the voice-note wire format.
// The upload pipeline (getUploadLocations -> AES-256-CBC -> octet-stream PUT
// -> ContentObject reference -> CreateContentMessage) is shared verbatim
// with the proven image/video sender.

// mp4AudioDurationMs validates the box structure of an MP4 file and returns
// the duration of its first audio track (mdhd timescale/duration). It reports
// whether any video track is present so callers can enforce audio-only voice
// notes. Unlike mp4MediaInfo this does not require a video track.
func mp4AudioDurationMs(data []byte) (durationMs int64, hasVideo bool, err error) {
	if len(data) < 12 || string(data[4:8]) != "ftyp" {
		return 0, false, fmt.Errorf("invalid MP4 container: missing ftyp header")
	}
	moov, err := mp4FindBox(data, 0, len(data), "moov")
	if err != nil {
		return 0, false, err
	}
	foundAudio := false
	offset := 8
	for offset+8 <= len(moov) {
		size, boxType, ok := mp4BoxHeader(moov, offset)
		if !ok {
			return 0, false, fmt.Errorf("malformed MP4: bad box header inside moov")
		}
		if boxType == "trak" {
			w, h, audio, durMs, err := mp4ParseTrak(moov[offset+8 : offset+size])
			if err != nil {
				return 0, false, err
			}
			if w > 0 && h > 0 {
				hasVideo = true
			} else if audio && !foundAudio {
				foundAudio = true
				durationMs = durMs
			}
		}
		offset += size
	}
	if !foundAudio {
		return 0, hasVideo, fmt.Errorf("malformed MP4: no audio track")
	}
	return durationMs, hasVideo, nil
}

// classifyOutgoingAudio classifies an outbound voice note. Only audio/mp4
// (the experimental first format) is accepted; every other container or an
// MP4 carrying a video track is rejected cleanly without transcoding.
func classifyOutgoingAudio(data []byte, declaredMIME string) (*OutgoingMediaInfo, error) {
	if !isMP4Data(data) {
		return nil, fmt.Errorf("outgoing voice notes support only audio/mp4 (experimental); declared %q is not supported", declaredMIME)
	}
	switch declaredMIME {
	case "", "audio/mp4", "audio/m4a", "audio/x-m4a":
	default:
		return nil, fmt.Errorf("media bytes are audio/mp4 but declared %q", declaredMIME)
	}
	durationMs, hasVideo, err := mp4AudioDurationMs(data)
	if err != nil {
		return nil, err
	}
	if hasVideo {
		return nil, fmt.Errorf("voice notes must be audio-only MP4; the MP4 carries a video track (send it as video instead)")
	}
	if durationMs <= 0 {
		return nil, fmt.Errorf("malformed MP4: audio track has no usable duration")
	}
	return &OutgoingMediaInfo{
		MIME:       "audio/mp4",
		MediaType:  protos.MediaType_MEDIA_TYPE_AUDIO,
		HasAudio:   true,
		DurationMs: durationMs,
	}, nil
}

// classifyOutgoingMedia dispatches one outbound media payload: declared
// audio/* MIME goes through the voice-note classifier; everything else keeps
// the proven image/video classification untouched.
func classifyOutgoingMedia(data []byte, declaredMIME string) (*OutgoingMediaInfo, error) {
	if len(declaredMIME) > 6 && declaredMIME[:6] == "audio/" {
		return classifyOutgoingAudio(data, declaredMIME)
	}
	return classifyOutgoingImage(data, declaredMIME)
}

// encodeNoteContents encodes the Contents payload for one outbound voice note
// (ContentType NOTE): Contents.note(6) -> Note.note(1, audio oneof) ->
// AudioNote{note(1)=AudioNoteDetails, userLocale(3)}. The AES key/IV travel
// base64-encoded inside AudioNoteDetails.encryptionInfo(4), mirroring how the
// web upload pipeline injects the crypto key/IV pair into the note metadata.
// mediaDurationMs is the parsed audio-track duration; durationSeconds mirrors
// the web sender's durationInSec (truncated to seconds).
func encodeNoteContents(info *OutgoingMediaInfo, key, iv []byte) []byte {
	details := bytes.Join([][]byte{
		// type = AUDIO (4) in the note metadata type enum (web hW.AUDIO;
		// distinct from media.Metadata_MediaType where AUDIO=3).
		mediaProtoVarint(2, 4),
		mediaProtoBytes(4, bytes.Join([][]byte{
			mediaProtoString(1, base64.StdEncoding.EncodeToString(key)),
			mediaProtoString(2, base64.StdEncoding.EncodeToString(iv)),
		}, nil)),
		mediaProtoVarint(8, 1), // hasSound: the web sender always sets true
		mediaProtoBytes(12, mediaProtoVarint(3, uint64(info.DurationMs/1000))),
		mediaProtoVarint(13, uint64(info.DurationMs)),
	}, nil)
	audioNote := bytes.Join([][]byte{
		mediaProtoBytes(1, details),
		mediaProtoString(3, "en"), // userLocale; incoming captures carry "en"
	}, nil)
	// Contents.note(6) -> Note.note(1) = AudioNote (oneof case audio = 1).
	return mediaProtoBytes(6, mediaProtoBytes(1, audioNote))
}
