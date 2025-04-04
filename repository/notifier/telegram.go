package notifier

import (
	"bytes"
	"context"
	"fmt"

	"github.com/code19m/sentinel/entity"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/telegram"
)

type telegramNotifier struct {
	notifier    notify.Notifier
	environment string
}

func NewTelegramNotifier(token string, chatIDs []int64, environment string) (*telegramNotifier, error) {
	tg, err := telegram.New(token)
	if err != nil {
		return nil, fmt.Errorf("NewTelegramNotifier: %w", err)
	}
	tg.AddReceivers(chatIDs...)

	n := notify.New()
	n.UseServices(tg)

	return &telegramNotifier{
		notifier:    n,
		environment: environment,
	}, nil
}

func (tn *telegramNotifier) Notify(ctx context.Context, e entity.ErrorInfo) error {
	// Build the message title
	msgTitle := tn.buildMsgTitle()

	// Build the message body
	msgBody := tn.buildMsgBody(e)

	// Send the message
	err := tn.notifier.Send(ctx, msgTitle, msgBody)
	if err != nil {
		return fmt.Errorf("telegramNotifier.Notify: %w", err)
	}

	return nil
}

func (tn *telegramNotifier) buildMsgTitle() string {
	return "<b>❗ Error from Sentinel</b>\n"
}

func (tn *telegramNotifier) buildMsgBody(e entity.ErrorInfo) string {
	var buffer bytes.Buffer

	// Main error information
	buffer.WriteString(fmt.Sprintf("<b>🔍 Environment:</b> %s\n", escapeHtml(tn.environment)))
	buffer.WriteString(fmt.Sprintf("<b>🛠️ Service:</b> %s\n", escapeHtml(e.Service)))
	buffer.WriteString(fmt.Sprintf("<b>🔄 Operation:</b> %s\n", escapeHtml(e.Operation)))
	buffer.WriteString(fmt.Sprintf("<b>🏷️ Code:</b> %s\n", escapeHtml(e.Code)))
	buffer.WriteString(fmt.Sprintf("<b>💬 Message:</b> %s\n", escapeHtml((e.Message))))

	// Separator for Details section
	buffer.WriteString("\n<b>📋 <i>Additional details</i></b>\n")

	// Details section with only visible details
	for k, v := range e.Details {
		if v != "" {
			if len(v) > 1000 {
				v = v[:1000] + "..."
			}
			buffer.WriteString(fmt.Sprintf("<i>%s</i>: <code>%s</code>\n", escapeHtml(k), v))
		}
	}

	return buffer.String()
}
