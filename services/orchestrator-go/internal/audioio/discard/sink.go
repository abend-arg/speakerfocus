package discard

import (
	"context"

	"github.com/abend-arg/speakerfocus/services/orchestrator-go/internal/audio"
)

type Sink struct{}

func (s Sink) Open(ctx context.Context, format audio.Format) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return format.Validate()
}

func (s Sink) WriteChunk(ctx context.Context, chunk audio.Chunk) error {
	return ctx.Err()
}

func (s Sink) Close() error {
	return nil
}
