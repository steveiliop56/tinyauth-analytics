package main

import (
	_ "embed"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"
)

const pageSize = 10

//go:embed dashboard.html
var dashboardTemplate string

type instance struct {
	UUID     string `json:"uuid"`
	Version  string `json:"version"`
	LastSeen int    `json:"last_seen"`
}

type instancesResponse struct {
	Instances []instance `json:"instances"`
	Total     int        `json:"total"`
}

type dashboardData struct {
	TotalInstances  int
	MostUsedVersion string
	Instances       []instance
	MaxPages        int
	NextPage        int
}

type dashboardHandler struct {
	api             string
	totalInstances  int
	mostUsedVersion string
	maxPages        int
	pages           [][]instance
}

func getInstances(api string) (instancesResponse, error) {
	resp, err := http.Get(api + "/v1/instances/all")

	if err != nil {
		return instancesResponse{}, err
	}

	defer resp.Body.Close()

	var instancesResp instancesResponse

	err = json.NewDecoder(resp.Body).Decode(&instancesResp)

	if err != nil {
		return instancesResponse{}, err
	}

	return instancesResp, nil
}

func parseInstancesToPages(instances []instance, pageSize int) [][]instance {
	var pages [][]instance

	for len(pages) < len(instances)/pageSize {
		pages = append(pages, instances[len(pages):len(pages)+pageSize])
	}

	return pages
}

func getMostUsedVersion(instances []instance) string {
	versionCount := make(map[string]int)

	for _, instance := range instances {
		versionCount[instance.Version]++
	}

	mostUsedVersion := ""
	maxCount := 0

	for version, count := range versionCount {
		if count > maxCount {
			maxCount = count
			mostUsedVersion = version
		}
	}

	return mostUsedVersion
}

func bundleInstances(instances [][]instance, pages int) []instance {
	var bundled []instance

	for i := 0; i < pages && i < len(instances); i++ {
		bundled = append(bundled, instances[i]...)
	}

	return bundled
}

func NewDashboardHandler(api string) *dashboardHandler {
	return &dashboardHandler{
		api: api,
	}
}

func (h *dashboardHandler) Init() error {
	go func() {
		ticker := time.NewTicker(time.Duration(24) * time.Hour)
		defer ticker.Stop()

		for ; true; <-ticker.C {
			log.Print("refreshing dashboard data")
			err := h.loadData()
			if err != nil {
				log.Printf("failed to refresh dashboard data: %v", err)
			}
		}
	}()

	return nil
}

func (h *dashboardHandler) loadData() error {
	instancesResp, err := getInstances(h.api)

	if err != nil {
		return err
	}

	h.totalInstances = instancesResp.Total
	h.maxPages = instancesResp.Total / pageSize
	h.pages = parseInstancesToPages(instancesResp.Instances, pageSize)
	h.mostUsedVersion = getMostUsedVersion(instancesResp.Instances)

	return nil
}

func (h *dashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	page := 0
	query := r.URL.Query()

	if val, ok := query["page"]; ok && len(val) > 0 {
		var err error
		page, err = strconv.Atoi(val[0])
		if err != nil || page < 0 || page >= len(h.pages) {
			http.Error(w, "invalid page number", http.StatusBadRequest)
			return
		}
	}

	tmpl, err := template.New("dashboard").Parse(dashboardTemplate)
	if err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, dashboardData{
		TotalInstances:  h.totalInstances,
		MostUsedVersion: h.mostUsedVersion,
		Instances:       bundleInstances(h.pages, page+1),
		MaxPages:        h.maxPages,
		NextPage:        page + 1,
	})

	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
}

func main() {
	mux := http.NewServeMux()
	dashboardHandler := NewDashboardHandler("https://api.tinyauth.app")
	err := dashboardHandler.Init()
	if err != nil {
		log.Printf("failed to initialize dashboard handler: %v", err)
		return
	}
	mux.Handle("/", dashboardHandler)
	log.Print("starting dashboard server on :8080")
	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Printf("server error: %v", err)
	}
}
