package log

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

func TimingUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		logger := FromContext(ctx)
		logger.Info("method execution time", "method", info.FullMethod, "duration", duration.String())

		return resp, err
	}
}

func TimingStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		err := handler(srv, stream)

		duration := time.Since(start)
		logger := FromContext(stream.Context())
		logger.Info("method execution time", "method", info.FullMethod, "duration", duration.String())

		return err
	}
}
