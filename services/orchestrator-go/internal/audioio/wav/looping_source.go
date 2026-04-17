package wav

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audio"
)

type LoopingFileSource struct {
	Path          string
	ChunkDuration time.Duration
	Loops         int

	source           *WavFileSource
	format           audio.Format
	currentLoop      int
	frameIndexBase   uint64
	sampleOffsetBase uint64
}

func (s *LoopingFileSource) Open(ctx context.Context) (audio.Format, error) {
	if s.Loops < 0 {
		return audio.Format{}, fmt.Errorf("loops must be greater than or equal to zero")
	}

	format, err := s.openLoop(ctx)
	if err != nil {
		return audio.Format{}, err
	}

	s.format = format
	s.currentLoop = 1
	s.frameIndexBase = 0
	s.sampleOffsetBase = 0

	return format, nil
}

func (s *LoopingFileSource) ReadChunk(ctx context.Context) (audio.Chunk, error) {
	for {
		if s.source == nil {
			return audio.Chunk{}, fmt.Errorf("source is not open")
		}

		chunk, err := s.source.ReadChunk(ctx)
		if err == nil {
			chunk.FrameIndex += s.frameIndexBase
			chunk.SampleOffset += s.sampleOffsetBase
			return chunk, nil
		}
		if !errors.Is(err, io.EOF) {
			return audio.Chunk{}, err
		}
		if s.Loops > 0 && s.currentLoop >= s.Loops {
			return audio.Chunk{}, io.EOF
		}

		s.frameIndexBase += s.source.frameIndex
		s.sampleOffsetBase += s.source.sampleOffset
		if err := s.source.Close(); err != nil {
			return audio.Chunk{}, err
		}

		format, err := s.openLoop(ctx)
		if err != nil {
			return audio.Chunk{}, err
		}
		if format != s.format {
			return audio.Chunk{}, fmt.Errorf("looped WAV format changed: got %#v want %#v", format, s.format)
		}
		s.currentLoop++
	}
}

func (s *LoopingFileSource) Close() error {
	if s.source == nil {
		return nil
	}
	err := s.source.Close()
	s.source = nil
	return err
}

func (s *LoopingFileSource) openLoop(ctx context.Context) (audio.Format, error) {
	source := &WavFileSource{
		Path:          s.Path,
		ChunkDuration: s.ChunkDuration,
	}
	format, err := source.Open(ctx)
	if err != nil {
		return audio.Format{}, err
	}
	s.source = source
	return format, nil
}
