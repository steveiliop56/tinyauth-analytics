package main

import (
	"cmp"
	_ "embed"
	"html/template"
	"log"
	"net/http"
	"slices"

	"github.com/tinyauthapp/analytics/queries"
)

//go:embed dashboard.html
var dashboardTemplate string

type DashboardHandler struct {
	queries         *queries.Queries
	minTagInstances int
}

type versionStats struct {
	Total         int
	MostUsed      string
	VersionLabels []string
	VersionValues []int
}

func NewDashboardHandler(queries *queries.Queries, minTagInstances int) *DashboardHandler {
	return &DashboardHandler{
		queries:         queries,
		minTagInstances: minTagInstances,
	}
}

func (h *DashboardHandler) sortVersionStats(stats versionStats) versionStats {
	type versionKv struct {
		label string
		value int
	}

	versionKvs := make([]versionKv, len(stats.VersionLabels))

	for i, version := range stats.VersionLabels {
		versionKvs[i] = versionKv{version, stats.VersionValues[i]}
	}

	slices.SortStableFunc(versionKvs, func(a, b versionKv) int {
		return cmp.Compare(b.value, a.value)
	})

	stats.VersionLabels = make([]string, len(versionKvs))
	stats.VersionValues = make([]int, len(versionKvs))

	for i, kv := range versionKvs {
		stats.VersionLabels[i] = kv.label
		stats.VersionValues[i] = kv.value
	}

	return stats
}

func (h *DashboardHandler) compileVersionStats(instances []queries.Instance) versionStats {
	stats := make(map[string]int)
	total := 0

	for _, instance := range instances {
		stats[instance.Version]++
		total++
	}

	mostUsed := "unknown"
	maxCount := 0

	versionLabels := make([]string, 0, len(stats))
	versionValues := make([]int, 0, len(stats))

	for version, count := range stats {
		if count < h.minTagInstances {
			continue
		}
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
	versionStatsSorted := h.sortVersionStats(versionStats)

	tmpl, err := template.New("dashboard").Parse(dashboardTemplate)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, versionStatsSorted)

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}
