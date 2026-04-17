package audio

import (
	"context"
	"fmt"
	"time"
)

type SampleFormat string

const (
	SampleFormatS16LE SampleFormat = "s16le"
	SampleFormatF32LE SampleFormat = "f32le"
)

type Format struct {
	SampleRate     uint32
	Channels       uint16
	SampleFormat   SampleFormat
	BytesPerSample uint16
}

func (f Format) Validate() error {
	if f.SampleRate == 0 {
		return fmt.Errorf("sample rate must be greater than zero")
	}
	if f.Channels == 0 {
		return fmt.Errorf("channels must be greater than zero")
	}
	if f.SampleFormat == "" {
		return fmt.Errorf("sample format is required")
	}
	if f.BytesPerSample == 0 {
		return fmt.Errorf("bytes per sample must be greater than zero")
	}

	switch f.SampleFormat {
	case SampleFormatS16LE:
		if f.BytesPerSample != 2 {
			return fmt.Errorf("sample format %s requires 2 bytes per sample", f.SampleFormat)
		}
	case SampleFormatF32LE:
		if f.BytesPerSample != 4 {
			return fmt.Errorf("sample format %s requires 4 bytes per sample", f.SampleFormat)
		}
	default:
		return fmt.Errorf("unsupported sample format %q", f.SampleFormat)
	}

	return nil
}

func (f Format) BytesPerFrame() int {
	return int(f.Channels * f.BytesPerSample)
}

func (f Format) BitsPerSample() uint16 {
	return f.BytesPerSample * 8
}

func (f Format) BytesForDuration(d time.Duration) (int, error) {
	if err := f.Validate(); err != nil {
		return 0, err
	}
	if d <= 0 {
		return 0, fmt.Errorf("duration must be greater than zero")
	}

	frames := (uint64(f.SampleRate) * uint64(d)) / uint64(time.Second)
	if frames == 0 {
		frames = 1
	}

	return int(frames) * f.BytesPerFrame(), nil
}

type Chunk struct {
	Data         []byte
	FrameIndex   uint64
	SampleOffset uint64
	CapturedAt   time.Time
	Duration     time.Duration
}

type Source interface {
	Open(ctx context.Context) (Format, error)
	ReadChunk(ctx context.Context) (Chunk, error)
	Close() error
}

type Sink interface {
	Open(ctx context.Context, format Format) error
	WriteChunk(ctx context.Context, chunk Chunk) error
	Close() error
}
