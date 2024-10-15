package service

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/code19m/sentinel/config"
	"github.com/code19m/sentinel/entity"
	"github.com/code19m/sentinel/pb"
	"github.com/code19m/sentinel/repository"
	"github.com/google/uuid"
	"github.com/nikoksr/notify"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewSentinelService(
	cfg config.Config,
	log *slog.Logger,
	store repository.Store,
	notifier notify.Notifier,
) pb.SentinelServiceServer {
	return &service{
		cfg:      cfg,
		log:      log,
		store:    store,
		notifier: notifier,
	}
}

type service struct {
	cfg      config.Config
	log      *slog.Logger
	store    repository.Store
	notifier notify.Notifier
	pb.UnimplementedSentinelServiceServer
}

func (s *service) SendError(ctx context.Context, in *pb.ErrorInfo) (*emptypb.Empty, error) {
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

	err := s.store.Add(ctx, e)
	if err != nil {
		return nil, fmt.Errorf("service.SendError: %w", err)
	}

	go s.handleAlert(ctx, e)

	return &emptypb.Empty{}, nil
}

func (s *service) handleAlert(ctx context.Context, e entity.ErrorInfo) {
	ctx = context.Background()

	lastAlerted, err := s.store.FindLast(ctx, e.Service, e.Operation, true)
	if err != nil && err != repository.ErrNotFound {
		s.log.ErrorContext(ctx, fmt.Sprintf("service.handleAlert: %v", err))
		return
	}

	// Skip alerting if the last alert was sent less than AlertCooldownMinutes ago
	if err == nil && time.Since(lastAlerted.CreatedAt) < time.Minute*time.Duration(s.cfg.AlertCooldownMinutes) {
		return
	}

	err = s.notifier.Send(ctx, s.buildAlertTitle(), s.buildAlertMessage(e))
	if err != nil {
		s.log.ErrorContext(ctx, fmt.Sprintf("service.handleAlert: %v", err))
		return
	}

	e.Alerted = true
	err = s.store.Update(ctx, e)
	if err != nil {
		s.log.ErrorContext(ctx, fmt.Sprintf("service.handleAlert: %v", err))
		return
	}
}

// buildAlertTitle creates a title for the alert with project name.
func (s *service) buildAlertTitle() string {
	return fmt.Sprintf("<b>🏗️ Project:</b> %s\n", escape(s.cfg.ProjectName))
}

// buildAlertMessage creates a structured alert message using HTML formatting.
func (s *service) buildAlertMessage(e entity.ErrorInfo) string {
	var buffer bytes.Buffer

	// Header with essential information, properly escaped
	buffer.WriteString(fmt.Sprintf("<b>🛠️ Service:</b> %s\n", escape(e.Service)))
	buffer.WriteString(fmt.Sprintf("<b>🔄 Operation:</b> %s\n", escape(e.Operation)))

	// Separator
	buffer.WriteString("\n")

	// Main error information
	buffer.WriteString(fmt.Sprintf("<b>❗ Code:</b> %s\n", escape(e.Code)))
	buffer.WriteString(fmt.Sprintf("<b>💬 Message:</b> %s\n", escape((e.Message))))

	// Separator for Details section
	buffer.WriteString("\n<b>📋 <i>Additional details</i></b>\n")

	// Details section with only visible details
	for k, v := range e.Details {
		if slices.Contains(s.cfg.AlertVisibleDetails, k) {
			buffer.WriteString(fmt.Sprintf("<i>%s</i>: <code>%s</code>\n", escape(k), escape(v)))
		}
	}

	return buffer.String()
}

func escape(in string) string {
	return html.EscapeString(replaceNewlines(in))

}

func replaceNewlines(in string) string {
	return strings.ReplaceAll(in, "\n", "\\n")
}
