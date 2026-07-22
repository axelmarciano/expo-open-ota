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
	"github.com/jackc/pgx/v5/pgxpool"
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

func (s *PostgresAuditStore) PurgeBefore(ctx context.Context, cutoff time.Time, exportedOnly bool) (int64, error) {
	pgCutoff := pgtype.Timestamptz{Time: cutoff, Valid: true}
	if exportedOnly {
		commandTag, err := s.engine.Queries.PurgeExportedAuditLogEventsBefore(ctx, pgCutoff)
		if err != nil {
			return 0, fmt.Errorf("failed to purge exported audit events from database: %w", err)
		}
		return commandTag.RowsAffected(), nil
	}
	commandTag, err := s.engine.Queries.PurgeAuditLogEventsBefore(ctx, pgCutoff)
	if err != nil {
		return 0, fmt.Errorf("failed to purge audit events from database: %w", err)
	}
	return commandTag.RowsAffected(), nil
}

func (s *PostgresAuditStore) ListAfter(ctx context.Context, afterID int64, limit int) ([]Event, error) {
	rows, err := s.engine.Queries.ListAuditLogEventsAfter(ctx, pgdb.ListAuditLogEventsAfterParams{
		ID:    afterID,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list audit events for export from database: %w", err)
	}
	events := make([]Event, len(rows))
	for i, row := range rows {
		events[i] = eventFromRow(row)
	}
	return events, nil
}

func (s *PostgresAuditStore) ExportCursor(ctx context.Context) (int64, error) {
	cursor, err := s.engine.Queries.GetAuditExportCursor(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to read the audit export cursor from database: %w", err)
	}
	return cursor, nil
}

// exportAdvisoryLockID serializes archive exporters across replicas (see
// migrationAdvisoryLockID in internal/database/postgres for the convention).
const exportAdvisoryLockID = 823672943

// TryExportLock claims the "one exporter at a time" advisory lock. A session
// advisory lock belongs to the connection that took it, so it lives on a
// connection pinned from the pool for the whole export, never on the shared
// pool where every query may land on a different session.
func (s *PostgresAuditStore) TryExportLock(ctx context.Context) (func(), bool, error) {
	pool, isPool := s.engine.DB.(*pgxpool.Pool)
	if !isPool {
		// No pool means no session to pin: run unlocked. The cursor CAS keeps
		// concurrent exporters correct, the lock only spares duplicate uploads.
		return func() {}, true, nil
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, false, fmt.Errorf("failed to acquire a connection for the audit export lock: %w", err)
	}
	var locked bool
	if err := conn.QueryRow(ctx, "SELECT pg_try_advisory_lock($1)", exportAdvisoryLockID).Scan(&locked); err != nil {
		conn.Release()
		return nil, false, fmt.Errorf("failed to take the audit export lock: %w", err)
	}
	if !locked {
		conn.Release()
		return nil, false, nil
	}
	release := func() {
		// Background context: the unlock must run even after the tick's
		// timeout. A failed unlock must not return a still-locked session to
		// the pool (the lock would leak forever), so the session is closed
		// instead: ending it releases every advisory lock it holds.
		if _, err := conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", exportAdvisoryLockID); err != nil {
			_ = conn.Conn().Close(context.Background())
		}
		conn.Release()
	}
	return release, true, nil
}

func (s *PostgresAuditStore) AdvanceExportCursor(ctx context.Context, from int64, to int64) (bool, error) {
	commandTag, err := s.engine.Queries.AdvanceAuditExportCursor(ctx, pgdb.AdvanceAuditExportCursorParams{
		LastExportedID:   from,
		LastExportedID_2: to,
	})
	if err != nil {
		return false, fmt.Errorf("failed to advance the audit export cursor in database: %w", err)
	}
	return commandTag.RowsAffected() == 1, nil
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
