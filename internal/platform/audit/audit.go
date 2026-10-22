package audit

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Event struct {
	ActorUserID   *uint
	EventType     string
	AggregateType string
	AggregateID   string
	Metadata      map[string]any
	IPAddress     string
	UserAgent     string
	CreatedAt     time.Time
}

func Write(ctx context.Context, db *gorm.DB, event Event) error {
	createdAt := event.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	return db.WithContext(ctx).Table("audit_events").Create(map[string]any{
		"actor_user_id":  event.ActorUserID,
		"event_type":     event.EventType,
		"aggregate_type": event.AggregateType,
		"aggregate_id":   event.AggregateID,
		"metadata":       event.Metadata,
		"ip_address":     event.IPAddress,
		"user_agent":     event.UserAgent,
		"created_at":     createdAt,
	}).Error
}
