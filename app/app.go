package app

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"runtime"

	"github.com/code19m/sentinel/config"
	"github.com/code19m/sentinel/pb"
	"github.com/code19m/sentinel/repository/notifier"
	"github.com/code19m/sentinel/repository/store"
	"github.com/code19m/sentinel/server"
	"github.com/code19m/sentinel/usecase"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Server is the main sentinel application server.
// It can be used standalone or embedded into another application.
type Server struct {
	logger *slog.Logger
	cfg    config.Config

	pgConn *pgxpool.Pool
	grpc   *grpc.Server
}

// NewServer creates a new Server instance. It connects to the database,
// initializes the notifier, and sets up the gRPC server.
func NewServer(
	logger *slog.Logger,
	cfg config.Config,
) (*Server, error) {
	ctx := context.Background()

	pgConn, err := pgxpool.New(ctx, fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser, cfg.PostgresPassword, cfg.PostgresDatabase))
	if err != nil {
		return nil, fmt.Errorf("sentinel: connect to database: %w", err)
	}

	pgStore, err := store.NewPgStore(pgConn, cfg.PostgresSchema)
	if err != nil {
		pgConn.Close()
		return nil, fmt.Errorf("sentinel: create store: %w", err)
	}

	notifier, err := defineNotifier(cfg)
	if err != nil {
		pgConn.Close()
		return nil, fmt.Errorf("sentinel: create notifier: %w", err)
	}

	uc := usecase.New(logger, pgStore, notifier, cfg.AlertCooldownMinutes)

	sentinelServer := server.NewSentinelServer(cfg, logger, uc)

	grpcPanicRecoveryHandler := func(p any) (err error) {
		buf := new(bytes.Buffer)
		stack := make([]byte, 2048)
		stackSize := runtime.Stack(stack, true)
		buf.Write(stack[:stackSize])

		err = status.Errorf(codes.Internal, "%s", p)

		logger.ErrorContext(ctx, "Panic recovered", slog.Any("error", err), slog.Any("panic", p), slog.String("stack", buf.String()))
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(
				recovery.WithRecoveryHandler(grpcPanicRecoveryHandler))))

	pb.RegisterSentinelServiceServer(grpcServer, sentinelServer)
	reflection.Register(grpcServer)

	return &Server{
		logger: logger,
		cfg:    cfg,
		pgConn: pgConn,
		grpc:   grpcServer,
	}, nil
}

// Start begins serving gRPC requests. It blocks until the server is stopped.
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", s.cfg.GrpcHost, s.cfg.GrpcPort))
	if err != nil {
		return fmt.Errorf("sentinel: listen: %w", err)
	}

	s.logger.Info("Server started", slog.String("address", listener.Addr().String()))

	if err := s.grpc.Serve(listener); err != nil {
		return fmt.Errorf("sentinel: serve: %w", err)
	}

	return nil
}

// Stop gracefully stops the server and closes the database connection.
func (s *Server) Stop() {
	s.grpc.GracefulStop()
	s.pgConn.Close()
	s.logger.Info("Server stopped")
}

func defineNotifier(cfg config.Config) (notifier.Notifier, error) {
	if cfg.AlertDisable {
		return notifier.NewNoopNotifier(), nil
	}

	switch cfg.AlertProvider {

	case config.AlertProviderTelegram:
		return notifier.NewTelegramNotifier(cfg.TelegramBotToken, cfg.TelegramsChatIDs, cfg.Environment)

	case config.AlertProviderDiscord:
		return notifier.NewDiscordNotifier(cfg.DiscordBotToken, cfg.DiscordChannelIDs, cfg.Environment)

	default:
		return nil, fmt.Errorf("defineNotifier: invalid alert provider: %s", cfg.AlertProvider)
	}
}
