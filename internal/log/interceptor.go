package log

import (
	"context"

	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		attrs := []any{slog.String("method", info.FullMethod)}
		logger := slog.Default().With(attrs...)

		ctx = WithLogger(ctx, logger)
		return handler(ctx, req)
	}
}

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w wrappedStream) Context() context.Context {
	return w.ctx
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		attrs := []any{slog.String("method", info.FullMethod)}
		logger := slog.Default().With(attrs...)

		ctx := WithLogger(stream.Context(), logger)
		wrapped := &wrappedStream{ServerStream: stream, ctx: ctx}
		return handler(srv, wrapped)
	}
}
