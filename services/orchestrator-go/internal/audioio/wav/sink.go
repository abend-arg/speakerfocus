package wav

import (
	"context"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audio"
)

type WavFileSink struct {
	Path   string
	file   *os.File
	format audio.Format
	bytes  uint32
}

func (s *WavFileSink) Open(ctx context.Context, format audio.Format) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.Path == "" {
		return fmt.Errorf("output path is required")
	}
	if err := format.Validate(); err != nil {
		return err
	}

	file, err := os.Create(s.Path)
	if err != nil {
		return err
	}
	s.file = file
	s.format = format
	s.bytes = 0

	if err := writeWavHeader(s.file, s.format, 0); err != nil {
		_ = s.Close()
		return err
	}

	return nil
}

func (s *WavFileSink) WriteChunk(ctx context.Context, chunk audio.Chunk) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s.file == nil {
		return fmt.Errorf("sink is not open")
	}
	if len(chunk.Data)%s.format.BytesPerFrame() != 0 {
		return fmt.Errorf("chunk data size %d is not aligned to frame size %d", len(chunk.Data), s.format.BytesPerFrame())
	}

	n, err := s.file.Write(chunk.Data)
	s.bytes += uint32(n)
	if err != nil {
		return err
	}
	if n != len(chunk.Data) {
		return fmt.Errorf("short write: wrote %d of %d bytes", n, len(chunk.Data))
	}

	return nil
}

func (s *WavFileSink) Close() error {
	if s.file == nil {
		return nil
	}

	var closeErr error
	if _, err := s.file.Seek(0, 0); err != nil {
		closeErr = err
	} else if err := writeWavHeader(s.file, s.format, s.bytes); err != nil {
		closeErr = err
	}

	if err := s.file.Close(); closeErr == nil {
		closeErr = err
	}
	s.file = nil

	return closeErr
}

func writeWavHeader(file *os.File, format audio.Format, dataBytes uint32) error {
	bitsPerSample, err := bitsPerSampleForWavPCM(format)
	if err != nil {
		return err
	}
	blockAlign := uint16(format.BytesPerFrame())
	byteRate := format.SampleRate * uint32(blockAlign)

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], 36+dataBytes)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16)
	binary.LittleEndian.PutUint16(header[20:22], 1)
	binary.LittleEndian.PutUint16(header[22:24], format.Channels)
	binary.LittleEndian.PutUint32(header[24:28], format.SampleRate)
	binary.LittleEndian.PutUint32(header[28:32], byteRate)
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)
	binary.LittleEndian.PutUint16(header[34:36], bitsPerSample)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], dataBytes)

	n, err := file.Write(header)
	if err != nil {
		return err
	}
	if n != len(header) {
		return fmt.Errorf("short header write: wrote %d of %d bytes", n, len(header))
	}

	return nil
}

func bitsPerSampleForWavPCM(format audio.Format) (uint16, error) {
	if err := format.Validate(); err != nil {
		return 0, err
	}
	switch format.SampleFormat {
	case audio.SampleFormatS16LE:
		return 16, nil
	default:
		return 0, fmt.Errorf("unsupported WAV output sample format %q: only s16le PCM is supported", format.SampleFormat)
	}
}
