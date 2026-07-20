// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package audit

import (
	"context"
	"encoding/json"
	"expo-open-ota/internal/database"
	"expo-open-ota/internal/database/postgres/pgdb"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresAuditStore struct {
	engine *database.Engine
}

func NewPostgresAuditStore(engine *database.Engine) *PostgresAuditStore {
	return &PostgresAuditStore{engine: engine}
}

// marshalMetadata keeps a nil map from becoming SQL NULL (the column is NOT
// NULL, no metadata means '{}') and a non-serializable value from costing the
// whole entry: the annotation is dropped and logged, the event still lands.
func marshalMetadata(action Action, metadata map[string]any) []byte {
	if len(metadata) == 0 {
		return []byte("{}")
	}
	payload, err := json.Marshal(metadata)
	if err != nil {
		log.Printf("audit: dropping unserializable metadata on %q: %v", action, err)
		return []byte("{}")
	}
	return payload
}

func eventFromRow(row pgdb.AuditLogEvent) Event {
	event := Event{
		ID:            row.ID,
		OccurredAt:    row.OccurredAt.Time,
		ActorType:     ActorType(row.ActorType),
		ActorID:       row.ActorID,
		ActorDisplay:  row.ActorDisplay,
		Action:        Action(row.Action),
		TargetType:    row.TargetType,
		TargetID:      row.TargetID,
		TargetDisplay: row.TargetDisplay,
		Outcome:       Outcome(row.Outcome),
		IP:            row.Ip,
		UserAgent:     row.UserAgent,
	}
	if row.AppID != nil {
		event.AppID = *row.AppID
	}
	// A row whose metadata does not parse still lists: the entry's facts
	// matter more than its annotations.
	if len(row.Metadata) > 0 {
		var metadata map[string]any
		if err := json.Unmarshal(row.Metadata, &metadata); err == nil && len(metadata) > 0 {
			event.Metadata = metadata
		}
	}
	return event
}

func toPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func (s *PostgresAuditStore) Insert(ctx context.Context, event Event) (Event, error) {
	// The zero *string is SQL NULL, an account-level event without an app.
	var appID *string
	if event.AppID != "" {
		appID = &event.AppID
	}
	row, err := s.engine.Queries.InsertAuditLogEvent(ctx, pgdb.InsertAuditLogEventParams{
		ActorType:     string(event.ActorType),
		ActorID:       event.ActorID,
		ActorDisplay:  event.ActorDisplay,
		Action:        string(event.Action),
		TargetType:    event.TargetType,
		TargetID:      event.TargetID,
		TargetDisplay: event.TargetDisplay,
		AppID:         appID,
		Outcome:       string(event.Outcome),
		Ip:            event.IP,
		UserAgent:     event.UserAgent,
		Metadata:      marshalMetadata(event.Action, event.Metadata),
	})
	if err != nil {
		return Event{}, fmt.Errorf("failed to insert audit event in database: %w", err)
	}
	return eventFromRow(row), nil
}

func (s *PostgresAuditStore) List(ctx context.Context, params ListParams) ([]Event, error) {
	rows, err := s.engine.Queries.ListAuditLogEvents(ctx, pgdb.ListAuditLogEventsParams{
		ActorID:      params.ActorID,
		Action:       params.Action,
		AppID:        params.AppID,
		Outcome:      params.Outcome,
		OccurredFrom: toPgTimestamptz(params.From),
		OccurredTo:   toPgTimestamptz(params.To),
		BeforeID:     params.BeforeID,
		RowLimit:     int32(params.Limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list audit events from database: %w", err)
	}
	events := make([]Event, len(rows))
	for i, row := range rows {
		events[i] = eventFromRow(row)
	}
	return events, nil
}

func (s *PostgresAuditStore) Count(ctx context.Context, filters ListFilters) (int64, error) {
	count, err := s.engine.Queries.CountAuditLogEvents(ctx, pgdb.CountAuditLogEventsParams{
		ActorID:      filters.ActorID,
		Action:       filters.Action,
		AppID:        filters.AppID,
		Outcome:      filters.Outcome,
		OccurredFrom: toPgTimestamptz(filters.From),
		OccurredTo:   toPgTimestamptz(filters.To),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count audit events in database: %w", err)
	}
	return count, nil
}
