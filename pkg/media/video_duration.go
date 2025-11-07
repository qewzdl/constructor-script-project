package media

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"time"
)

type boxHeader struct {
	size      int64
	headerLen int64
	boxType   string
	start     int64
}

// MP4Duration parses the duration of an MP4 file located at the provided path.
// Only ISO Base Media (MP4/MOV) containers with an mvhd atom are supported.
func MP4Duration(path string) (time.Duration, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	return mp4DurationFromReader(file)
}

func mp4DurationFromReader(r io.ReadSeeker) (time.Duration, error) {
	fileSize, err := r.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	if _, err := r.Seek(0, io.SeekStart); err != nil {
		return 0, err
	}

	for {
		pos, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		if pos >= fileSize {
			break
		}

		header, err := readBoxHeader(r, fileSize)
		if err != nil {
			return 0, err
		}

		payload := header.size - header.headerLen
		switch header.boxType {
		case "moov":
			return parseMoovBox(r, header.start+header.size, payload)
		default:
			if _, err := r.Seek(payload, io.SeekCurrent); err != nil {
				return 0, err
			}
		}
	}

	return 0, fmt.Errorf("moov box not found in media file")
}

func readBoxHeader(r io.ReadSeeker, limit int64) (boxHeader, error) {
	start, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return boxHeader{}, err
	}

	var header boxHeader
	header.start = start

	var base [8]byte
	if _, err := io.ReadFull(r, base[:]); err != nil {
		return boxHeader{}, err
	}

	header.headerLen = 8
	header.boxType = string(base[4:8])

	size := binary.BigEndian.Uint32(base[:4])
	switch size {
	case 0:
		header.size = limit - start
	case 1:
		var large [8]byte
		if _, err := io.ReadFull(r, large[:]); err != nil {
			return boxHeader{}, err
		}
		header.headerLen += 8
		header.size = int64(binary.BigEndian.Uint64(large[:]))
	default:
		header.size = int64(size)
	}

	if header.size < header.headerLen {
		return boxHeader{}, fmt.Errorf("invalid box size for %s", header.boxType)
	}
	if header.start+header.size > limit {
		return boxHeader{}, fmt.Errorf("box %s exceeds parent bounds", header.boxType)
	}

	return header, nil
}

func parseMoovBox(r io.ReadSeeker, endOffset int64, payloadSize int64) (time.Duration, error) {
	for {
		pos, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return 0, err
		}
		if pos >= endOffset {
			break
		}

		header, err := readBoxHeader(r, endOffset)
		if err != nil {
			return 0, err
		}

		payload := header.size - header.headerLen
		if header.boxType == "mvhd" {
			return parseMvhdBox(r, payload)
		}

		if _, err := r.Seek(payload, io.SeekCurrent); err != nil {
			return 0, err
		}
	}

	return 0, fmt.Errorf("mvhd box not found in moov container")
}

func parseMvhdBox(r io.ReadSeeker, payloadSize int64) (time.Duration, error) {
	if payloadSize < 4 {
		return 0, fmt.Errorf("mvhd box too small")
	}

	var versionAndFlags [4]byte
	if _, err := io.ReadFull(r, versionAndFlags[:]); err != nil {
		return 0, err
	}
	version := versionAndFlags[0]
	consumed := int64(4)

	switch version {
	case 0:
		if payloadSize-consumed < 16 {
			return 0, fmt.Errorf("mvhd payload too small for version 0")
		}
		var data [16]byte
		if _, err := io.ReadFull(r, data[:]); err != nil {
			return 0, err
		}
		consumed += 16

		timescale := binary.BigEndian.Uint32(data[8:12])
		duration := binary.BigEndian.Uint32(data[12:16])
		if timescale == 0 {
			return 0, fmt.Errorf("mvhd timescale is zero")
		}

		if payloadSize > consumed {
			if _, err := r.Seek(payloadSize-consumed, io.SeekCurrent); err != nil {
				return 0, err
			}
		}

		return scaleDuration(float64(duration), float64(timescale)), nil
	case 1:
		if payloadSize-consumed < 28 {
			return 0, fmt.Errorf("mvhd payload too small for version 1")
		}
		var data [28]byte
		if _, err := io.ReadFull(r, data[:]); err != nil {
			return 0, err
		}
		consumed += 28

		timescale := binary.BigEndian.Uint32(data[16:20])
		duration := binary.BigEndian.Uint64(data[20:28])
		if timescale == 0 {
			return 0, fmt.Errorf("mvhd timescale is zero")
		}

		if payloadSize > consumed {
			if _, err := r.Seek(payloadSize-consumed, io.SeekCurrent); err != nil {
				return 0, err
			}
		}

		return scaleDuration(float64(duration), float64(timescale)), nil
	default:
		return 0, fmt.Errorf("unsupported mvhd version %d", version)
	}
}

func scaleDuration(duration, timescale float64) time.Duration {
	if timescale <= 0 {
		return 0
	}
	seconds := duration / timescale
	if seconds <= 0 {
		return 0
	}
	return time.Duration(seconds * float64(time.Second))
}
