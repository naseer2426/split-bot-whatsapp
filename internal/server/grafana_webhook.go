package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/naseer2426/split-bot-whatsapp/internal/config"
)

// Grafana webhook payload structures
type grafanaAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL,omitempty"`
}

type grafanaWebhookPayload struct {
	Receiver string         `json:"receiver"`
	Status   string         `json:"status"`
	Alerts   []grafanaAlert `json:"alerts"`
}

// grafanaWebhookHandler handles POST requests from Grafana alert webhooks
func (s *Server) grafanaWebhookHandler(c *gin.Context) {
	var payload grafanaWebhookPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	if len(payload.Alerts) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No alerts in payload",
		})
		return
	}

	groupID := config.Get().Grafana.AlertGroupID
	if groupID == "" {
		if groupIDFromLabel, ok := payload.Alerts[0].Labels["group_id"]; ok {
			groupID = groupIDFromLabel
		} else if groupIDFromAnnotation, ok := payload.Alerts[0].Annotations["group_id"]; ok {
			groupID = groupIDFromAnnotation
		}
	}

	if groupID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "group_id not found. Set GRAFANA_ALERT_GROUP_ID environment variable or include 'group_id' in alert labels/annotations",
		})
		return
	}

	message := formatGrafanaAlertMessage(payload)

	if err := s.handler.SendMessageToGroup(message, groupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send message",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Grafana alert notification sent successfully",
	})
}

// formatGrafanaAlertMessage formats Grafana alert payload into a readable WhatsApp message
func formatGrafanaAlertMessage(payload grafanaWebhookPayload) string {
	var builder strings.Builder

	gmtPlus8, err := time.LoadLocation("Asia/Singapore")
	if err != nil {
		gmtPlus8 = time.FixedZone("GMT+8", 8*60*60)
	}

	builder.WriteString(fmt.Sprintf("🚨 *Grafana Alert: %s*\n\n", strings.ToUpper(payload.Status)))

	for i, alert := range payload.Alerts {
		if i > 0 {
			builder.WriteString("\n---\n\n")
		}

		if alertName, ok := alert.Labels["alertname"]; ok {
			builder.WriteString(fmt.Sprintf("*Alert:* %s\n", alertName))
		}

		builder.WriteString(fmt.Sprintf("*Status:* %s\n", alert.Status))

		if description, ok := alert.Annotations["description"]; ok {
			builder.WriteString(fmt.Sprintf("*Description:* %s\n", description))
		}

		if summary, ok := alert.Annotations["summary"]; ok {
			builder.WriteString(fmt.Sprintf("*Summary:* %s\n", summary))
		}

		excludedLabels := map[string]bool{
			"alertname": true,
			"severity":  true,
		}
		hasAdditionalLabels := false
		for key, value := range alert.Labels {
			if !excludedLabels[key] {
				if !hasAdditionalLabels {
					builder.WriteString("\n*Details:*\n")
					hasAdditionalLabels = true
				}
				builder.WriteString(fmt.Sprintf("• %s: %s\n", key, value))
			}
		}

		if alert.StartsAt != "" {
			if t, err := time.Parse(time.RFC3339, alert.StartsAt); err == nil {
				tGMT8 := t.In(gmtPlus8)
				builder.WriteString(fmt.Sprintf("\n*Started:* %s\n", tGMT8.Format("2006-01-02 15:04:05 MST")))
			}
		}

		if alert.Status == "resolved" && alert.EndsAt != "" {
			if t, err := time.Parse(time.RFC3339, alert.EndsAt); err == nil {
				tGMT8 := t.In(gmtPlus8)
				builder.WriteString(fmt.Sprintf("*Resolved:* %s\n", tGMT8.Format("2006-01-02 15:04:05 MST")))
			}
		}

		if alert.GeneratorURL != "" {
			builder.WriteString(fmt.Sprintf("\n*View:* %s\n", alert.GeneratorURL))
		}
	}

	return builder.String()
}
