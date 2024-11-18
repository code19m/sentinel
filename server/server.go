package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/code19m/sentinel/config"
	"github.com/code19m/sentinel/entity"
	"github.com/code19m/sentinel/pb"
	"github.com/code19m/sentinel/usecase"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewSentinelServer(
	cfg config.Config,
	log *slog.Logger,
	usecase usecase.UseCase,
) pb.SentinelServiceServer {
	return &server{
		cfg:     cfg,
		log:     log,
		usecase: usecase,
	}
}

type server struct {
	cfg     config.Config
	log     *slog.Logger
	usecase usecase.UseCase
	pb.UnimplementedSentinelServiceServer
}

func (s *server) SendError(ctx context.Context, in *pb.ErrorInfo) (*emptypb.Empty, error) {
	e := entity.ErrorInfo{
		ID:        uuid.New().String(),
		Code:      in.GetCode(),
		Message:   in.GetMessage(),
		Details:   in.GetDetails(),
		Service:   in.GetService(),
		Operation: in.GetOperation(),
		CreatedAt: time.Now(),
		Alerted:   false,
	}

	err := s.usecase.SendError(ctx, e)
	if err != nil {
		s.log.ErrorContext(ctx, fmt.Sprintf("server.SendError: %v", err))
		return nil, fmt.Errorf("server.SendError: %w", err)
	}

	return &emptypb.Empty{}, nil
}
