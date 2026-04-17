package main

import (
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/tinyauthapp/analytics/queries"
)

//go:embed dashboard.html
var dashboardTemplate string

//go:embed favicon.ico
var faviconData []byte

type DashboardHandler struct {
	queries *queries.Queries
}

type versionStats struct {
	Total         int
	MostUsed      string
	VersionLabels []string
	VersionValues []int
}

func NewDashboardHandler(queries *queries.Queries) *DashboardHandler {
	return &DashboardHandler{
		queries: queries,
	}
}

func (h *DashboardHandler) compileVersionStats(instances []queries.Instance) versionStats {
	stats := make(map[string]int)
	total := 0

	for _, instance := range instances {
		stats[instance.Version]++
		total++
	}

	mostUsed := ""
	maxCount := 0

	versionLabels := make([]string, 0, len(stats))
	versionValues := make([]int, 0, len(stats))

	for version, count := range stats {
		if count > maxCount {
			maxCount = count
			mostUsed = version
		}
		versionLabels = append(versionLabels, version)
		versionValues = append(versionValues, count)
	}

	return versionStats{
		Total:         total,
		MostUsed:      mostUsed,
		VersionLabels: versionLabels,
		VersionValues: versionValues,
	}
}

func (h *DashboardHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	instances, err := h.queries.GetAllInstances(r.Context())

	if err != nil {
		log.Printf("failed to get instances: %v", err)
		http.Error(w, "Failed to retrieve instances", http.StatusInternalServerError)
		return
	}

	versionStats := h.compileVersionStats(instances)

	fmt.Println(versionStats)

	tmpl, err := template.New("dashboard").Parse(dashboardTemplate)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, versionStats)

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func (h *DashboardHandler) Favicon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/x-icon")
	w.WriteHeader(http.StatusOK)
	w.Write(faviconData)
}

func (h *DashboardHandler) Robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("User-agent: *\nDisallow: /"))
}
