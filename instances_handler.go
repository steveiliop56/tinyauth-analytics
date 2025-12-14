package main

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"time"
	"tinyauth-analytics/database/queries"

	"github.com/go-chi/render"
)

type InstancesHandler struct {
	queries *queries.Queries
}

func NewInstancesHandler(queries *queries.Queries) *InstancesHandler {
	return &InstancesHandler{
		queries: queries,
	}
}

func (h *InstancesHandler) GetInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := h.queries.GetAllInstances(r.Context())

	if err != nil {
		slog.Error("failed to get instances", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"status":  "500",
			"message": "Failed to retrieve instances",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, map[string]any{
		"status":    "200",
		"instances": instances,
		"total":     len(instances),
	})
}

func (h *InstancesHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var heartbeat struct {
		UUID    string `json:"uuid"`
		Version string `json:"version"`
	}

	err := render.DecodeJSON(r.Body, &heartbeat)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"status":  "400",
			"message": "Invalid request payload",
		})
		return
	}

	_, err = h.queries.GetInstance(r.Context(), heartbeat.UUID)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		slog.Error("failed to get instance", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"status":  "500",
			"message": "Failed to retrieve instance",
		})
		return
	}

	if err != nil && errors.Is(err, sql.ErrNoRows) {
		err = h.queries.CreateInstance(r.Context(), queries.CreateInstanceParams{
			UUID:     heartbeat.UUID,
			Version:  heartbeat.Version,
			LastSeen: time.Now().UnixMilli(),
		})
		if err != nil {
			slog.Error("failed to create instance", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			render.JSON(w, r, map[string]string{
				"status":  "500",
				"message": "Failed to create instance",
			})
			return
		}
		w.WriteHeader(http.StatusCreated)
		render.JSON(w, r, map[string]string{
			"status":  "201",
			"message": "Instance created",
		})
		return
	}

	err = h.queries.UpdateInstance(r.Context(), queries.UpdateInstanceParams{
		LastSeen: time.Now().UnixMilli(),
		UUID:     heartbeat.UUID,
	})

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			"status":  "500",
			"message": "Failed to update instance",
		})
		return
	}

	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, map[string]string{
		"status":  "200",
		"message": "Instance updated",
	})
}
