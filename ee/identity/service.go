// Copyright (c) 2026 Axel Marciano (Mercure Technologies). All rights reserved.
// This file is governed by the Mercure Technologies Enterprise Edition License
// (see ee/LICENSE); it is NOT covered by the MIT license of this repository.

package identity

import (
	"context"
	"fmt"
	"time"
)

// Op is one identity operation as carried on the wire (the log event name).
// The vocabulary mirrors PostHog/Mixpanel/Amplitude on purpose: $set merges,
// $set_once only fills absent keys, $unset removes. There is deliberately no
// reset(): the eas_client_id cannot rotate, so logout is expressed as $unset.
type Op string

const (
	OpSet     Op = "$set"
	OpSetOnce Op = "$set_once"
	OpUnset   Op = "$unset"
)

// IsIdentityOp reports whether a log event name is one of the identity
// operations; the ingest route uses it to route identity events away from the
// telemetry path.
func IsIdentityOp(eventName string) bool {
	switch Op(eventName) {
	case OpSet, OpSetOnce, OpUnset:
		return true
	}
	return false
}

// IdentityMutator is what the service needs from the store; narrow on purpose
// so tests can fake it and future stores stay honest about the contract.
type IdentityMutator interface {
	ApplySet(ctx context.Context, appID string, easClientID string, raw map[string]any, geo *Geo) (ApplyResult, error)
	ApplySetOnce(ctx context.Context, appID string, easClientID string, raw map[string]any, geo *Geo) (ApplyResult, error)
	ApplyUnset(ctx context.Context, appID string, easClientID string, keys []string, geo *Geo) (ApplyResult, error)
}

// Service applies identity operations coming off the wire: resolve the
// request IP into a geo enrichment, dispatch to the store, and account for
// what happened in Prometheus. The ingest route owns transport concerns
// (decoding, response codes); the service owns semantics.
type Service struct {
	store IdentityMutator
	geo   GeoResolver
}

// NewService builds the identity service. geo may be nil: identity works
// without a GeoLite2 database, devices simply stay unlocated.
func NewService(store IdentityMutator, geo GeoResolver) *Service {
	return &Service{store: store, geo: geo}
}

// Request is one identity operation extracted from a log event.
type Request struct {
	AppID       string
	EASClientID string
	Op          Op
	// Attributes carries the key/value payload of $set and $set_once.
	Attributes map[string]any
	// UnsetKeys carries the key names of $unset.
	UnsetKeys []string
	// RemoteIP is the already-resolved client IP of the HTTP request that
	// delivered the batch (proxy handling happens upstream).
	RemoteIP string
}

func (s *Service) Apply(ctx context.Context, req Request) (ApplyResult, error) {
	start := time.Now()

	var geo *Geo
	if s.geo != nil && req.RemoteIP != "" {
		geo = s.geo.Resolve(req.RemoteIP)
	}

	var result ApplyResult
	var err error
	switch req.Op {
	case OpSet:
		result, err = s.store.ApplySet(ctx, req.AppID, req.EASClientID, req.Attributes, geo)
	case OpSetOnce:
		result, err = s.store.ApplySetOnce(ctx, req.AppID, req.EASClientID, req.Attributes, geo)
	case OpUnset:
		result, err = s.store.ApplyUnset(ctx, req.AppID, req.EASClientID, req.UnsetKeys, geo)
	default:
		// The op sentinel keeps the label set bounded: req.Op is wire input
		// here, not one of our constants.
		err = fmt.Errorf("unknown identity op %q", req.Op)
		observeApply(req.AppID, Op("unknown"), err, 0, time.Since(start))
		return ApplyResult{}, err
	}

	observeApply(req.AppID, req.Op, err, len(result.DroppedKeys), time.Since(start))
	if err != nil {
		return ApplyResult{}, err
	}
	return result, nil
}
