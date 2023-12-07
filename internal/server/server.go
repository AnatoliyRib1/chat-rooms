package server

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/cloudmachinery/chat-rooms/tree/AnatoliyRib1/main/internal/errlog"

	"github.com/redis/go-redis/v9"

	"github.com/cloudmachinery/chat-rooms/tree/AnatoliyRib1/main/apis/chat"
	"github.com/cloudmachinery/chat-rooms/tree/AnatoliyRib1/main/internal/config"
	"github.com/cloudmachinery/chat-rooms/tree/AnatoliyRib1/main/internal/log"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	cfg         *config.Config
	grpcServer  *grpc.Server
	closers     []func() error
	listener    net.Listener
	redisClient *redis.Client
}

func New(cfg *config.Config) (*Server, error) {
	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			log.UnaryServerInterceptor(),
			errlog.UnaryServerInterceptor(),
			log.TimingUnaryInterceptor(),
		),
		grpc.ChainStreamInterceptor(
			log.StreamServerInterceptor(),
			errlog.StreamServerInterceptor(),
			log.TimingStreamInterceptor(),
		),
	)
	rdb, err := getRedisClient(context.Background(), cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	s := NewStore(rdb)
	h := NewChatServer(s)
	chat.RegisterChatServiceServer(grpcServer, h)
	reflection.Register(grpcServer)

	return &Server{
		cfg:         cfg,
		grpcServer:  grpcServer,
		redisClient: rdb,
	}, nil
}

func (s *Server) Start() error {
	logger, err := log.SetupLogger(s.cfg.Local, s.cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("setup log: %s", err)
	}
	slog.SetDefault(logger)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.cfg.Port))
	if err != nil {
		return fmt.Errorf("listen: %s", err)
	}
	s.listener = lis
	s.closers = append(s.closers, lis.Close)

	s.closers = append(s.closers, s.redisClient.Close)

	logger.Info("starting server", "port", s.cfg.Port)
	return s.grpcServer.Serve(lis)
}

func (s *Server) Stop(ctx context.Context) error {
	stopped := make(chan struct{})

	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	select {
	case <-ctx.Done():
		s.grpcServer.Stop()
	case <-stopped:

	}
	return withClosers(s.closers, nil)
}

func (s *Server) Port() (int, error) {
	if s.listener == nil || s.listener.Addr() == nil {
		return 0, errors.New("server is not started")
	}
	return s.listener.Addr().(*net.TCPAddr).Port, nil
}

func withClosers(closers []func() error, err error) error {
	errs := []error{err}

	for i := len(closers) - 1; i >= 0; i-- {
		if err = closers[i](); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func getRedisClient(ctx context.Context, redisURL string) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: redisURL,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return rdb, nil
}
