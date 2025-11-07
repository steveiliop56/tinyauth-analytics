package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Variables
var version = "development"

//go:embed dashboard.html
var dashboardTemplate string

//go:embed favicon.ico
var faviconData []byte

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
	apiServer       string
	pageSize        int
	refreshInterval int
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

	for pageSize < len(instances) {
		instances, pages = instances[pageSize:], append(pages, instances[0:pageSize])
	}

	pages = append(pages, instances)
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

func NewDashboardHandler(apiServer string, pageSize int, refreshInterval int) *dashboardHandler {
	return &dashboardHandler{
		apiServer:       apiServer,
		pageSize:        pageSize,
		refreshInterval: refreshInterval,
	}
}

func (h *dashboardHandler) Init() error {
	go func() {
		ticker := time.NewTicker(time.Duration(h.refreshInterval) * time.Minute)
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
	res, err := getInstances(h.apiServer)

	if err != nil {
		return err
	}

	h.totalInstances = res.Total
	h.maxPages = res.Total / h.pageSize
	h.pages = parseInstancesToPages(res.Instances, h.pageSize)
	h.mostUsedVersion = getMostUsedVersion(res.Instances)

	if strings.TrimSpace(h.mostUsedVersion) == "" {
		h.mostUsedVersion = "N/A"
	}

	log.Printf("loaded %d instances, most used version: %s", h.totalInstances, h.mostUsedVersion)

	return nil
}

func (h *dashboardHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("received request for", r.URL.Path)
	switch r.URL.Path {
	case "/":
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
		// No indexing for the analytics dashboard
	case "/robots.txt":
		w.Write([]byte("User-agent: *\nDisallow: /"))
	case "/favicon.ico":
		w.Header().Set("Content-Type", "image/x-icon")
		w.Write(faviconData)
	default:
		http.NotFound(w, r)
	}
}

func main() {
	log.Printf("tinyauth analytics dashboard version %s", version)

	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	address := os.Getenv("ADDRESS")

	if address == "" {
		address = "0.0.0.0"
	}

	apiServer := os.Getenv("API_SERVER")

	if apiServer == "" {
		apiServer = "https://api.tinyauth.app"
	}

	pageSize := 10
	pageSizeEnv := os.Getenv("PAGE_SIZE")

	if pageSizeEnv != "" {
		ps, err := strconv.Atoi(pageSizeEnv)
		if err != nil {
			log.Printf("invalid PAGE_SIZE value, using default %d: %v", pageSize, err)
		} else {
			pageSize = ps
		}
	}

	refreshInterval := 30
	refreshIntervalEnv := os.Getenv("REFRESH_INTERVAL")

	if refreshIntervalEnv != "" {
		ps, err := strconv.Atoi(refreshIntervalEnv)
		if err != nil {
			log.Printf("invalid REFRESH_INTERVAL value, using default %d: %v", refreshInterval, err)
		} else {
			refreshInterval = ps
		}
	}

	mux := http.NewServeMux()

	dashboardHandler := NewDashboardHandler(apiServer, pageSize, refreshInterval)
	err := dashboardHandler.Init()
	if err != nil {
		log.Printf("failed to initialize dashboard handler: %v", err)
		return
	}

	mux.Handle("/", dashboardHandler)

	bind := fmt.Sprintf("%s:%s", address, port)

	log.Printf("starting server on %s", bind)
	err = http.ListenAndServe(bind, mux)
	if err != nil {
		log.Printf("server error: %v", err)
	}
}
