package wav

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audio"
)

type WavFileSource struct {
	Path          string
	ChunkDuration time.Duration

	file          *os.File
	format        audio.Format
	chunkBytes    int
	dataRemaining uint32
	frameIndex    uint64
	sampleOffset  uint64
}

func (s *WavFileSource) Open(ctx context.Context) (audio.Format, error) {
	if err := ctx.Err(); err != nil {
		return audio.Format{}, err
	}
	if s.Path == "" {
		return audio.Format{}, fmt.Errorf("input path is required")
	}
	if s.ChunkDuration <= 0 {
		return audio.Format{}, fmt.Errorf("chunk duration must be greater than zero")
	}

	file, err := os.Open(s.Path)
	if err != nil {
		return audio.Format{}, err
	}
	s.file = file

	format, dataSize, err := readWavHeader(file)
	if err != nil {
		_ = s.Close()
		return audio.Format{}, err
	}

	chunkBytes, err := format.BytesForDuration(s.ChunkDuration)
	if err != nil {
		_ = s.Close()
		return audio.Format{}, err
	}

	s.format = format
	s.chunkBytes = chunkBytes
	s.dataRemaining = dataSize
	s.frameIndex = 0
	s.sampleOffset = 0

	return s.format, nil
}

func (s *WavFileSource) ReadChunk(ctx context.Context) (audio.Chunk, error) {
	if err := ctx.Err(); err != nil {
		return audio.Chunk{}, err
	}
	if s.file == nil {
		return audio.Chunk{}, fmt.Errorf("source is not open")
	}
	if s.dataRemaining == 0 {
		return audio.Chunk{}, io.EOF
	}

	capturedAt := time.Now()
	bytesToRead := s.chunkBytes
	if uint32(bytesToRead) > s.dataRemaining {
		bytesToRead = int(s.dataRemaining)
	}

	data := make([]byte, bytesToRead)
	n, err := io.ReadFull(s.file, data)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return audio.Chunk{}, err
	}
	if n == 0 {
		return audio.Chunk{}, io.EOF
	}

	data = data[:n]
	s.dataRemaining -= uint32(n)

	frames := uint64(n / s.format.BytesPerFrame())
	chunk := audio.Chunk{
		Data:         data,
		FrameIndex:   s.frameIndex,
		SampleOffset: s.sampleOffset,
		CapturedAt:   capturedAt,
		Duration:     framesToDuration(frames, s.format.SampleRate),
	}

	s.frameIndex++
	s.sampleOffset += frames

	return chunk, nil
}

func (s *WavFileSource) Close() error {
	if s.file == nil {
		return nil
	}
	err := s.file.Close()
	s.file = nil
	return err
}

func readWavHeader(r io.ReadSeeker) (audio.Format, uint32, error) {
	var riff [12]byte
	if _, err := io.ReadFull(r, riff[:]); err != nil {
		return audio.Format{}, 0, err
	}
	if string(riff[0:4]) != "RIFF" || string(riff[8:12]) != "WAVE" {
		return audio.Format{}, 0, fmt.Errorf("unsupported WAV header")
	}

	var format *audio.Format
	var dataSize uint32

	for {
		var chunkHeader [8]byte
		if _, err := io.ReadFull(r, chunkHeader[:]); err != nil {
			return audio.Format{}, 0, fmt.Errorf("failed to find WAV fmt/data chunks: %w", err)
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return audio.Format{}, 0, fmt.Errorf("invalid WAV fmt chunk size %d", chunkSize)
			}
			buf := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, buf); err != nil {
				return audio.Format{}, 0, err
			}

			audioFormat := binary.LittleEndian.Uint16(buf[0:2])
			if audioFormat != 1 {
				return audio.Format{}, 0, fmt.Errorf("unsupported WAV encoding %d: only PCM is supported", audioFormat)
			}

			bitsPerSample := binary.LittleEndian.Uint16(buf[14:16])
			sampleFormat, bytesPerSample, err := sampleFormatFromWavPCM(bitsPerSample)
			if err != nil {
				return audio.Format{}, 0, err
			}

			f := audio.Format{
				Channels:       binary.LittleEndian.Uint16(buf[2:4]),
				SampleRate:     binary.LittleEndian.Uint32(buf[4:8]),
				SampleFormat:   sampleFormat,
				BytesPerSample: bytesPerSample,
			}
			if err := f.Validate(); err != nil {
				return audio.Format{}, 0, err
			}
			blockAlign := binary.LittleEndian.Uint16(buf[12:14])
			if int(blockAlign) != f.BytesPerFrame() {
				return audio.Format{}, 0, fmt.Errorf("WAV block align %d does not match frame size %d", blockAlign, f.BytesPerFrame())
			}
			format = &f

		case "data":
			dataSize = chunkSize
			if format == nil {
				return audio.Format{}, 0, fmt.Errorf("WAV data chunk appears before fmt chunk")
			}
			if dataSize%uint32(format.BytesPerFrame()) != 0 {
				return audio.Format{}, 0, fmt.Errorf("WAV data size %d is not aligned to frame size %d", dataSize, format.BytesPerFrame())
			}
			return *format, dataSize, nil

		default:
			if _, err := r.Seek(int64(chunkSize), io.SeekCurrent); err != nil {
				return audio.Format{}, 0, err
			}
		}

		if chunkSize%2 == 1 {
			if _, err := r.Seek(1, io.SeekCurrent); err != nil {
				return audio.Format{}, 0, err
			}
		}
	}
}

func sampleFormatFromWavPCM(bitsPerSample uint16) (audio.SampleFormat, uint16, error) {
	switch bitsPerSample {
	case 16:
		return audio.SampleFormatS16LE, 2, nil
	default:
		return "", 0, fmt.Errorf("unsupported WAV PCM bit depth %d: only 16-bit PCM is supported", bitsPerSample)
	}
}

func framesToDuration(frames uint64, sampleRate uint32) time.Duration {
	return time.Duration((frames * uint64(time.Second)) / uint64(sampleRate))
}
