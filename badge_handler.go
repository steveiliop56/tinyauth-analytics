package main

import (
	"fmt"
	"net/http"

	_ "embed"

	"github.com/go-chi/render"
	"github.com/tinyauthapp/analytics/queries"
)

type BadgeHandler struct {
	queries *queries.Queries
}

func NewBadgeHandler(queries *queries.Queries) *BadgeHandler {
	return &BadgeHandler{
		queries: queries,
	}
}

type shieldsioData struct {
	SchemaVersion int    `json:"schemaVersion"`
	Label         string `json:"label"`
	Message       string `json:"message"`
	Color         string `json:"color,omitempty"`
	LabelColor    string `json:"labelColor,omitempty"`
	IsError       bool   `json:"isError,omitempty"`
	NamedLogo     string `json:"namedLogo,omitempty"`
	LogoSvg       string `json:"logoSvg,omitempty"`
	LogoColor     string `json:"logoColor,omitempty"`
	LogoSize      string `json:"logoSize,omitempty"`
	Style         string `json:"style,omitempty"`
}

func (h *BadgeHandler) Badge(w http.ResponseWriter, r *http.Request) {
	instanceCount, err := h.queries.GetInstanceCount(r.Context())

	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	badgeData := shieldsioData{
		SchemaVersion: 1,
		Label:         "Active Instances",
		Message:       fmt.Sprintf("%d", instanceCount),
		Color:         "brightgreen",
		LabelColor:    "grey",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	render.JSON(w, r, badgeData)
}
