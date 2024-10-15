package main

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"runtime"

	"github.com/code19m/sentinel/config"
	"github.com/code19m/sentinel/pb"
	"github.com/code19m/sentinel/repository"
	"github.com/code19m/sentinel/service"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/telegram"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

func main() {
	ctx := context.Background()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.ErrorContext(ctx, "Failed to start service", slog.Any("error", err))
		os.Exit(1)
	}

	pgConn, err := pgxpool.New(ctx, fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDatabase))

	if err != nil {
		logger.ErrorContext(ctx, "Failed to connect to database", slog.Any("error", err))
		os.Exit(1)
	}

	repo, err := repository.NewPgStore(pgConn)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to create repository", slog.Any("error", err))
		os.Exit(1)
	}

	tg, err := telegram.New(cfg.TelegramBotToken)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to create telegram service", slog.Any("error", err))
		os.Exit(1)
	}
	tg.AddReceivers(cfg.TelegramsRecipients...)

	notifier := notify.New()
	notifier.UseServices(tg)

	service := service.NewSentinelService(cfg, logger, repo, notifier)

	grpcPanicRecoveryHandler := func(p any) (err error) {
		buf := new(bytes.Buffer)
		stack := make([]byte, 2048)             // Allocate a byte slice for the stack trace
		stackSize := runtime.Stack(stack, true) // Capture the stack trace
		buf.Write(stack[:stackSize])            // Write the stack trace to the buffer

		err = status.Errorf(codes.Internal, "%s", p)

		logger.ErrorContext(ctx, "Panic recovered", slog.Any("error", err), slog.Any("panic", p), slog.String("stack", buf.String()))
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(
				recovery.WithRecoveryHandler(grpcPanicRecoveryHandler))))

	// Register service
	pb.RegisterSentinelServiceServer(grpcServer, service)
	reflection.Register(grpcServer)

	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", cfg.GrpcHost, cfg.GrpcPort))
	if err != nil {
		logger.ErrorContext(ctx, "Failed to listen", slog.Any("error", err))
		os.Exit(1)
	}

	logger.InfoContext(ctx, "Server started", slog.String("address", listener.Addr().String()))

	err = grpcServer.Serve(listener)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to serve", slog.Any("error", err))
		os.Exit(1)
	}
}
