package snapapi

import (
	"encoding/binary"
	"fmt"
)

// mp4MediaInfo parses the minimum metadata the EXTERNAL_MEDIA video send
// needs: display dimensions (from the video track's tkhd: 16.16 fixed point
// width/height, corrected by the tkhd transformation matrix so rotated
// portrait phone videos report their display size), media duration in
// milliseconds (mdhd timescale/duration of the video track), and audio
// presence (any mdia/hdlr handler type "soun"). It validates the box
// structure (ftyp header, moov, one video track) without decoding the media,
// mirroring how webpDimensions validates WebP structurally.
func mp4MediaInfo(data []byte) (width, height int, hasAudio bool, durationMs int64, err error) {
	if len(data) < 12 || string(data[4:8]) != "ftyp" {
		return 0, 0, false, 0, fmt.Errorf("invalid MP4 container: missing ftyp header")
	}
	moov, err := mp4FindBox(data, 0, len(data), "moov")
	if err != nil {
		return 0, 0, false, 0, err
	}
	foundVideo := false
	// Walk moov children: trak boxes carry tkhd (dims) and mdia/hdlr (handlers).
	offset := 8
	for offset+8 <= len(moov) {
		size, boxType, ok := mp4BoxHeader(moov, offset)
		if !ok {
			return 0, 0, false, 0, fmt.Errorf("malformed MP4: bad box header inside moov")
		}
		if boxType == "trak" {
			w, h, audio, durMs, err := mp4ParseTrak(moov[offset+8 : offset+size])
			if err != nil {
				return 0, 0, false, 0, err
			}
			if w > 0 && h > 0 {
				if foundVideo {
					// Prefer the first video track; only audio presence accumulates.
					if audio {
						hasAudio = true
					}
				} else {
					width, height, foundVideo = w, h, true
					hasAudio = audio
					durationMs = durMs
				}
			} else if audio {
				hasAudio = true
			}
		}
		offset += size
	}
	if !foundVideo {
		return 0, 0, false, 0, fmt.Errorf("malformed MP4: no video track with usable dimensions")
	}
	return width, height, hasAudio, durationMs, nil
}

// mp4ParseTrak extracts display dimensions from tkhd (trailing 16.16 fixed
// point width/height, corrected by the tkhd transformation matrix), the media
// duration from mdhd (timescale/duration), and the handler type from
// mdia/hdlr.
func mp4ParseTrak(trak []byte) (width, height int, hasAudio bool, durationMs int64, err error) {
	offset := 0
	var timescale, mediaDuration uint32
	for offset+8 <= len(trak) {
		size, boxType, ok := mp4BoxHeader(trak, offset)
		if !ok {
			return 0, 0, false, 0, fmt.Errorf("malformed MP4: bad box header inside trak")
		}
		payload := trak[offset+8 : offset+size]
		switch boxType {
		case "tkhd":
			// Width/height are the trailing 4+4 bytes as 16.16 fixed point
			// (84 bytes for version 0, 96 for version 1).
			if len(payload) < 8 {
				return 0, 0, false, 0, fmt.Errorf("malformed MP4: tkhd too short")
			}
			codedW := int(binary.BigEndian.Uint32(payload[len(payload)-8:]) >> 16)
			codedH := int(binary.BigEndian.Uint32(payload[len(payload)-4:]) >> 16)
			// The 36-byte transformation matrix sits directly before them.
			// Rotated portrait phone videos carry coded (unrotated) dims with
			// a 90/270-degree matrix; recipients expect display dimensions.
			width, height = mp4DisplayDimensions(codedW, codedH, payload)
		case "mdia":
			handler, ts, dur, err := mp4ParseMdia(payload)
			if err != nil {
				return 0, 0, false, 0, err
			}
			timescale, mediaDuration = ts, dur
			switch handler {
			case "soun":
				hasAudio = true
			case "vide":
				// Video dimensions come from tkhd.
			default:
				// Other tracks (metadata, hint) are ignored.
			}
		}
		offset += size
	}
	if timescale > 0 && mediaDuration > 0 {
		durationMs = int64(mediaDuration) * 1000 / int64(timescale)
	}
	return width, height, hasAudio, durationMs, nil
}

// mp4ParseMdia walks the mdia children for hdlr (handler type) and mdhd
// (timescale + media duration).
func mp4ParseMdia(mdia []byte) (handler string, timescale, mediaDuration uint32, err error) {
	offset := 0
	for offset+8 <= len(mdia) {
		size, boxType, ok := mp4BoxHeader(mdia, offset)
		if !ok {
			return "", 0, 0, fmt.Errorf("malformed MP4: bad box header inside mdia")
		}
		payload := mdia[offset+8 : offset+size]
		switch boxType {
		case "hdlr":
			if len(payload) < 12 {
				return "", 0, 0, fmt.Errorf("malformed MP4: hdlr too short")
			}
			handler = string(payload[8:12])
		case "mdhd":
			if len(payload) >= 20 {
				if payload[0] == 1 && len(payload) >= 32 {
					timescale = binary.BigEndian.Uint32(payload[20:24])
					// 64-bit duration: clamp to the low word.
					mediaDuration = binary.BigEndian.Uint32(payload[28:32])
				} else {
					timescale = binary.BigEndian.Uint32(payload[12:16])
					mediaDuration = binary.BigEndian.Uint32(payload[16:20])
				}
			}
		}
		offset += size
	}
	return handler, timescale, mediaDuration, nil
}

// mp4DisplayDimensions applies the tkhd transformation matrix (16.16 fixed
// point, 36 bytes directly before the trailing width/height) to the coded
// track dimensions. A 90/270-degree rotation matrix swaps them; degenerate
// or zeroed matrices fall back to the coded dimensions.
func mp4DisplayDimensions(codedW, codedH int, tkhd []byte) (int, int) {
	if len(tkhd) < 44 || codedW <= 0 || codedH <= 0 {
		return codedW, codedH
	}
	m := tkhd[len(tkhd)-44 : len(tkhd)-8]
	a := int32(binary.BigEndian.Uint32(m[0:4]))
	b := int32(binary.BigEndian.Uint32(m[4:8]))
	c := int32(binary.BigEndian.Uint32(m[12:16]))
	d := int32(binary.BigEndian.Uint32(m[16:20]))
	if a == 0 && b == 0 && c == 0 && d == 0 {
		return codedW, codedH
	}
	w := int((absI32(a)*int64(codedW) + absI32(c)*int64(codedH)) >> 16)
	h := int((absI32(b)*int64(codedW) + absI32(d)*int64(codedH)) >> 16)
	if w <= 0 || h <= 0 {
		return codedW, codedH
	}
	return w, h
}

func absI32(v int32) int64 {
	if v < 0 {
		return -int64(v)
	}
	return int64(v)
}

// mp4FindBox locates the first box of the given type between from (inclusive
// header) and to, returning its full payload (header excluded).
func mp4FindBox(data []byte, from, to int, boxType string) ([]byte, error) {
	offset := from
	for offset+8 <= to {
		size, bType, ok := mp4BoxHeader(data, offset)
		if !ok {
			return nil, fmt.Errorf("malformed MP4: bad box header at %d", offset)
		}
		if offset+size > to {
			return nil, fmt.Errorf("truncated MP4: box %q extends past available bytes", bType)
		}
		if bType == boxType {
			return data[offset : offset+size], nil
		}
		offset += size
	}
	return nil, fmt.Errorf("malformed MP4: missing %q box", boxType)
}

// mp4BoxHeader reads the 4-byte big-endian size (including the 8-byte header)
// and 4-byte type at offset. Zero-size and 64-bit largesize boxes are
// rejected; ordinary chat MP4s never need them.
func mp4BoxHeader(data []byte, offset int) (size int, boxType string, ok bool) {
	if offset+8 > len(data) {
		return 0, "", false
	}
	size = int(binary.BigEndian.Uint32(data[offset : offset+4]))
	if size < 8 || offset+size > len(data) {
		return 0, "", false
	}
	return size, string(data[offset+4 : offset+8]), true
}

func isMP4Data(data []byte) bool {
	return len(data) >= 12 && string(data[4:8]) == "ftyp"
}
