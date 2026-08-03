// Package events berisi adapter publikasi domain event approval.
package events

import (
	"log/slog"

	"github.com/gsnpeeps/gsnpeeps/backend/internal/domain"
)

// LoggingPublisher mencatat intent notifikasi tanpa menyimpannya. Persistence notifikasi
// yang idempotent dikerjakan pada epic BE.5; adapter ini menjaga agar side effect tidak
// hilang diam-diam sekaligus membuat boundary event dapat diuji sekarang.
//
// Event hanya memuat identifier dan aktor, tanpa alasan, dokumen, atau data personal lain.
type LoggingPublisher struct {
	logger *slog.Logger
}

func NewLoggingPublisher(logger *slog.Logger) *LoggingPublisher {
	return &LoggingPublisher{logger: logger}
}

func (p *LoggingPublisher) Publish(events ...domain.ApprovalEvent) error {
	for _, event := range events {
		attributes := []any{
			"event", string(event.Type),
			"request_id", event.RequestID,
			"status", string(event.Status),
		}
		if event.NextStage != nil {
			attributes = append(attributes, "next_stage", string(*event.NextStage))
		}
		if event.ActorUserID == nil {
			attributes = append(attributes, "actor", "system")
		}
		p.logger.Info("approval event emitted", attributes...)
	}
	return nil
}
