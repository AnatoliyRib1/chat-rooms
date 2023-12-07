package errlog

import (
	"context"

	"github.com/cloudmachinery/chat-rooms/tree/AnatoliyRib1/main/internal/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		res, err := handler(ctx, req)
		if err != nil {

			if ifCanceledOrDeadlineExceeded(err) {
				return res, nil
			}
			logger := log.FromContext(ctx)
			logger.Error("error happened", "error", err)
		}
		return res, nil
	}
}

type wrappedStream struct {
	grpc.ServerStream
}

func (w wrappedStream) SendMsg(msg any) error {
	err := w.ServerStream.SendMsg(msg)
	if err != nil {
		if ifCanceledOrDeadlineExceeded(err) {
			return nil
		}
		logger := log.FromContext(w.Context())
		logger.Error("error happened", "error", err)
	}
	return err
}

func (w wrappedStream) RecvMsg(msg any) error {
	err := w.ServerStream.RecvMsg(msg)
	if err != nil {
		if ifCanceledOrDeadlineExceeded(err) {
			return nil
		}
		logger := log.FromContext(w.Context())
		logger.Error("error happened", "error", err)
	}
	return err
}

func StreamServerInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, stream grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapped := &wrappedStream{ServerStream: stream}
		err := handler(srv, wrapped)
		if err != nil {
			if ifCanceledOrDeadlineExceeded(err) {
				return nil
			}
			logger := log.FromContext(stream.Context())
			logger.Error("error happened", "error", err)
		}
		return err
	}
}

func ifCanceledOrDeadlineExceeded(err error) bool {
	return status.Code(err) == codes.Canceled || status.Code(err) == codes.DeadlineExceeded
}
