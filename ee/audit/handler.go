package audit

import (
	"encoding/json"
	"expo-open-ota/internal/handlers"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type AuditHandler struct {
	service *AuditService
}

func NewAuditHandler(service *AuditService) *AuditHandler {
	return &AuditHandler{service: service}
}

func optStr(v string) *string {
	if v == "" {
		return nil
	}
	return &v
}

func checkAndParseLimit(limit *string) int {
	if limit == nil {
		return 0
	}
	if n, err := strconv.Atoi(*limit); err == nil {
		return n
	}
	return 0
}

func checkAndParseBeforeID(beforeID *string) *int64 {
	if beforeID == nil {
		return nil
	}
	if n, err := strconv.ParseInt(*beforeID, 10, 64); err == nil {
		return &n
	}
	return nil
}

func checkAndParseRange(from *string, to *string) (error, *time.Time, *time.Time) {
	if from == nil && to == nil {
		return nil, nil, nil
	}
	var finalFrom *(time.Time)
	var finalTo *(time.Time)

	if from != nil {
		parsedFrom, errFrom := time.Parse(time.RFC3339, *from)
		if errFrom != nil {
			return errFrom, nil, nil
		}
		finalFrom = &parsedFrom
	}
	if to != nil {
		parsedTo, errTo := time.Parse(time.RFC3339, *to)
		if errTo != nil {
			return errTo, nil, nil
		}
		finalTo = &parsedTo
	}
	if finalTo != nil && finalFrom != nil {
		notValid := finalTo.Before(*finalFrom)
		if notValid {
			return fmt.Errorf("%s must be before %s", finalFrom, finalTo), nil, nil
		}
	}
	return nil, finalFrom, finalTo
}

func renderJSON(w http.ResponseWriter, status int, payload interface{}) {
	marshaledResponse, _ := json.Marshal(payload)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(marshaledResponse)
}

func (h *AuditHandler) ListAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	errRange, from, to := checkAndParseRange(
		optStr(query.Get("From")),
		optStr(query.Get("To")),
	)
	if errRange != nil {
		handlers.RenderError(w, http.StatusBadRequest, errRange.Error())
		return
	}
	params := ListParams{
		ListFilters: ListFilters{
			ActorID: optStr(query.Get("ActorID")),
			Action:  optStr(query.Get("Action")),
			AppID:   optStr(query.Get("AppId")),
			Outcome: optStr(query.Get("Outcome")),
			From:    from,
			To:      to,
		},
		BeforeID: checkAndParseBeforeID(optStr(query.Get("BeforeID"))),
		Limit:    checkAndParseLimit(optStr(query.Get("Limit"))),
	}

	events, nextCursor, err := h.service.List(r.Context(), params)
	if err != nil {
		fmt.Printf("Failed to list audit logs %s", err.Error())
		handlers.RenderError(w, http.StatusBadRequest, "Failed to list audit logs")
		return
	}
	count, err := h.service.Count(r.Context(), params.ListFilters)
	if err != nil {
		fmt.Printf("Failed to count audit logs %s", err.Error())
		handlers.RenderError(w, http.StatusBadRequest, "Failed to count audit logs")
		return
	}
	renderJSON(w, http.StatusOK, map[string]interface{}{
		"events":     events,
		"nextCursor": nextCursor,
		"count":      count,
	})
}
