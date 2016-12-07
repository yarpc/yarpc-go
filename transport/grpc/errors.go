package grpc

import (
	"fmt"

	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
)

type errCantExtractHeader struct {
	Name string
	MD   metadata.MD
}

func (e errCantExtractHeader) Error() string {
	return fmt.Sprintf("could not extract header %q from context metadata (%v)", e.Name, e.MD)
}

type errCantExtractMetadata struct {
	ctx context.Context
}

func (e errCantExtractMetadata) Error() string {
	return fmt.Sprintf("could not extract metadata information from context (%v)", e.ctx)
}
