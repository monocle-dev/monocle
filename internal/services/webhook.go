package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/monocle-dev/monocle/internal/models"
)

type DiscordWebhookField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type DiscordEmbed struct {
	Title       string                `json:"title"`
	Description string                `json:"description"`
	Color       int                   `json:"color"`
	Fields      []DiscordWebhookField `json:"fields"`
	Footer      *DiscordFooter        `json:"footer,omitempty"`
	Timestamp   string                `json:"timestamp"`
}

type DiscordFooter struct {
	Text string `json:"text"`
}

type DiscordWebhookRequest struct {
	Username  string         `json:"username"`
	AvatarURL string         `json:"avatar_url,omitempty"`
	Embeds    []DiscordEmbed `json:"embeds"`
}

type SlackField struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

type SlackAttachment struct {
	Color     string       `json:"color"`
	Title     string       `json:"title"`
	Text      string       `json:"text"`
	Fields    []SlackField `json:"fields"`
	Footer    string       `json:"footer"`
	Timestamp int64        `json:"ts"`
}

type SlackWebhookRequest struct {
	Username    string            `json:"username"`
	IconEmoji   string            `json:"icon_emoji,omitempty"`
	Text        string            `json:"text"`
	Attachments []SlackAttachment `json:"attachments"`
}

const (
	ColorRed    = 16711680 // #FF0000 - Incident created
	ColorGreen  = 65280    // #00FF00 - Incident resolved
	ColorOrange = 16753920 // #FFA500 - Warning

	Username  = "Monocle Monitor"
	AvatarURL = "https://avatars.githubusercontent.com/u/219688397"
)

func SendIncidentCreatedNotification(project models.Project, incident models.Incident) error {
	if project.DiscordWebhook != "" {
		if err := sendDiscordIncidentCreated(project.DiscordWebhook, project, incident); err != nil {
			return fmt.Errorf("discord: %w", err)
		}
	}

	if project.SlackWebhook != "" {
		if err := sendSlackIncidentCreated(project.SlackWebhook, project, incident); err != nil {
			return fmt.Errorf("slack: %w", err)
		}
	}

	return nil
}

func SendIncidentResolvedNotification(project models.Project, incident models.Incident) error {
	if project.DiscordWebhook != "" {
		if err := sendDiscordIncidentResolved(project.DiscordWebhook, project, incident); err != nil {
			return fmt.Errorf("discord: %w", err)
		}
	}

	if project.SlackWebhook != "" {
		if err := sendSlackIncidentResolved(project.SlackWebhook, project, incident); err != nil {
			return fmt.Errorf("slack: %w", err)
		}
	}

	return nil
}

func sendDiscordIncidentCreated(webhookURL string, project models.Project, incident models.Incident) error {
	startedAt := "Unknown"
	if incident.StartedAt != nil {
		startedAt = incident.StartedAt.Format("2006-01-02 15:04:05 UTC")
	}

	payload := DiscordWebhookRequest{
		Username:  Username,
		AvatarURL: AvatarURL,
		Embeds: []DiscordEmbed{
			{
				Title:       "🚨 **INCIDENT DETECTED**",
				Description: fmt.Sprintf("**%s** has encountered an issue and requires attention.", incident.Monitor.Name),
				Color:       ColorRed,
				Fields: []DiscordWebhookField{
					{Name: "📊 Monitor", Value: incident.Monitor.Name, Inline: true},
					{Name: "🏷️ Monitor Type", Value: incident.Monitor.Type, Inline: true},
					{Name: "⚠️ Status", Value: "**" + incident.Status + "**", Inline: true},
					{Name: "📝 Incident Title", Value: incident.Title, Inline: false},
					{Name: "📋 Description", Value: incident.Description, Inline: false},
					{Name: "⏰ Started At", Value: startedAt, Inline: true},
					{Name: "🔄 Check Interval", Value: fmt.Sprintf("%d seconds", incident.Monitor.Interval), Inline: true},
				},
				Footer: &DiscordFooter{
					Text: fmt.Sprintf("Project: %s | Monocle Monitoring", project.Name),
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	return sendDiscordWebhook(webhookURL, payload)
}

func sendDiscordIncidentResolved(webhookURL string, project models.Project, incident models.Incident) error {
	startedAt := "Unknown"
	resolvedAt := "Unknown"
	duration := "Unknown"

	if incident.StartedAt != nil {
		startedAt = incident.StartedAt.Format("2006-01-02 15:04:05 UTC")
	}

	if incident.ResolvedAt != nil {
		resolvedAt = incident.ResolvedAt.Format("2006-01-02 15:04:05 UTC")
		if incident.StartedAt != nil {
			duration = incident.ResolvedAt.Sub(*incident.StartedAt).Round(time.Second).String()
		}
	}

	payload := DiscordWebhookRequest{
		Username:  Username,
		AvatarURL: AvatarURL,
		Embeds: []DiscordEmbed{
			{
				Title:       "✅ **INCIDENT RESOLVED**",
				Description: fmt.Sprintf("**%s** is back to normal operation.", incident.Monitor.Name),
				Color:       ColorGreen,
				Fields: []DiscordWebhookField{
					{Name: "📊 Monitor", Value: incident.Monitor.Name, Inline: true},
					{Name: "🏷️ Monitor Type", Value: incident.Monitor.Type, Inline: true},
					{Name: "✅ Status", Value: "**" + incident.Status + "**", Inline: true},
					{Name: "📝 Incident Title", Value: incident.Title, Inline: false},
					{Name: "⏰ Started At", Value: startedAt, Inline: true},
					{Name: "🏁 Resolved At", Value: resolvedAt, Inline: true},
					{Name: "⏱️ Duration", Value: duration, Inline: true},
				},
				Footer: &DiscordFooter{
					Text: fmt.Sprintf("Project: %s | Monocle Monitoring", project.Name),
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	return sendDiscordWebhook(webhookURL, payload)
}

func sendSlackIncidentCreated(webhookURL string, project models.Project, incident models.Incident) error {
	startedAt := "Unknown"
	if incident.StartedAt != nil {
		startedAt = incident.StartedAt.Format("2006-01-02 15:04:05 UTC")
	}

	payload := SlackWebhookRequest{
		Username:  Username,
		IconEmoji: ":rotating_light:",
		Text:      ":rotating_light: *INCIDENT DETECTED*",
		Attachments: []SlackAttachment{
			{
				Color: "danger",
				Title: fmt.Sprintf("Monitor '%s' has encountered an issue", incident.Monitor.Name),
				Text:  incident.Description,
				Fields: []SlackField{
					{Title: "Monitor", Value: incident.Monitor.Name, Short: true},
					{Title: "Type", Value: incident.Monitor.Type, Short: true},
					{Title: "Status", Value: incident.Status, Short: true},
					{Title: "Interval", Value: fmt.Sprintf("%d seconds", incident.Monitor.Interval), Short: true},
					{Title: "Incident Title", Value: incident.Title, Short: false},
					{Title: "Started At", Value: startedAt, Short: false},
				},
				Footer:    fmt.Sprintf("Project: %s", project.Name),
				Timestamp: time.Now().Unix(),
			},
		},
	}

	return sendSlackWebhook(webhookURL, payload)
}

func sendSlackIncidentResolved(webhookURL string, project models.Project, incident models.Incident) error {
	startedAt := "Unknown"
	resolvedAt := "Unknown"
	duration := "Unknown"

	if incident.StartedAt != nil {
		startedAt = incident.StartedAt.Format("2006-01-02 15:04:05 UTC")
	}

	if incident.ResolvedAt != nil {
		resolvedAt = incident.ResolvedAt.Format("2006-01-02 15:04:05 UTC")
		if incident.StartedAt != nil {
			duration = incident.ResolvedAt.Sub(*incident.StartedAt).Round(time.Second).String()
		}
	}

	payload := SlackWebhookRequest{
		Username:  Username,
		IconEmoji: ":white_check_mark:",
		Text:      ":white_check_mark: *INCIDENT RESOLVED*",
		Attachments: []SlackAttachment{
			{
				Color: "good",
				Title: fmt.Sprintf("Monitor '%s' is back to normal operation", incident.Monitor.Name),
				Text:  "The incident has been resolved and the monitor is functioning normally.",
				Fields: []SlackField{
					{Title: "Monitor", Value: incident.Monitor.Name, Short: true},
					{Title: "Type", Value: incident.Monitor.Type, Short: true},
					{Title: "Status", Value: incident.Status, Short: true},
					{Title: "Duration", Value: duration, Short: true},
					{Title: "Incident Title", Value: incident.Title, Short: false},
					{Title: "Started At", Value: startedAt, Short: true},
					{Title: "Resolved At", Value: resolvedAt, Short: true},
				},
				Footer:    fmt.Sprintf("Project: %s", project.Name),
				Timestamp: time.Now().Unix(),
			},
		},
	}

	return sendSlackWebhook(webhookURL, payload)
}

func sendDiscordWebhook(webhookURL string, payload DiscordWebhookRequest) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord payload: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send Discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Discord webhook returned status %d", resp.StatusCode)
	}

	return nil
}

func sendSlackWebhook(webhookURL string, payload SlackWebhookRequest) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack payload: %w", err)
	}

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to send Slack webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("Slack webhook returned status %d", resp.StatusCode)
	}

	return nil
}
