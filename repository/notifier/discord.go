package notifier

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/code19m/sentinel/entity"
	"github.com/nikoksr/notify"
	"github.com/nikoksr/notify/service/discord"
)

type discordNotifier struct {
	notifier            notify.Notifier
	environment         string
	alertVisibleDetails []string
	showAllDetails      bool
}

func NewDiscordNotifier(token string, channelIDs []string, environment string, alertVisibleDetails []string) (*discordNotifier, error) {
	d := discord.New()
	err := d.AuthenticateWithBotToken(token)
	if err != nil {
		return nil, fmt.Errorf("NewDiscordNotifier: %w", err)
	}
	d.AddReceivers(channelIDs...)

	n := notify.New()
	n.UseServices(d)

	return &discordNotifier{
		notifier:            n,
		environment:         environment,
		alertVisibleDetails: alertVisibleDetails,
		showAllDetails:      len(alertVisibleDetails) == 1 && alertVisibleDetails[0] == "*",
	}, nil
}

func (dn *discordNotifier) Notify(ctx context.Context, e entity.ErrorInfo) error {
	// Build the message title
	msgTitle := dn.buildMsgTitle()

	// Build the message body
	msgBody := dn.buildMsgBody(e)

	// Send the message
	err := dn.notifier.Send(ctx, msgTitle, msgBody)
	if err != nil {
		return fmt.Errorf("discordNotifier.Notify: %w", err)
	}

	return nil
}

func (dn *discordNotifier) buildMsgTitle() string {
	return "**❗ Error from Sentinel**\n"
}

func (dn *discordNotifier) buildMsgBody(e entity.ErrorInfo) string {
	var buffer bytes.Buffer

	// Main error information
	buffer.WriteString(fmt.Sprintf("**🔍 Environment:** %s\n", escapeMarkdown(dn.environment)))
	buffer.WriteString(fmt.Sprintf("**🛠️ Service:** %s\n", escapeMarkdown(e.Service)))
	buffer.WriteString(fmt.Sprintf("**🔄 Operation:** %s\n", escapeMarkdown(e.Operation)))
	buffer.WriteString(fmt.Sprintf("**🏷️ Code:** %s\n", escapeMarkdown(e.Code)))
	buffer.WriteString(fmt.Sprintf("**💬 Message:** %s\n", escapeMarkdown(e.Message)))

	// Separator for Details section
	buffer.WriteString("\n**📋 _Additional details_**\n")

	// Details section with only visible details
	for k, v := range e.Details {
		if slices.Contains(dn.alertVisibleDetails, k) || dn.showAllDetails {
			buffer.WriteString(fmt.Sprintf("_%s_: `%s`\n", escapeMarkdown(k), escapeMarkdown(v)))
		}
	}

	return buffer.String()
}
