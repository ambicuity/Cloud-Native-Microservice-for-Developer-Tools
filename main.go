package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// BuildService represents our microservice
type BuildService struct {
	db      DatabaseInterface
	metrics *Metrics
}

// BuildRequest represents a build request
type BuildRequest struct {
	ID          int       `json:"id" db:"id"`
	ProjectName string    `json:"project_name" db:"project_name"`
	GitURL      string    `json:"git_url" db:"git_url"`
	Branch      string    `json:"branch" db:"branch"`
	Status      string    `json:"status" db:"status"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Metrics holds prometheus metrics
type Metrics struct {
	BuildsTotal     prometheus.CounterVec
	BuildDuration   prometheus.HistogramVec
	ActiveBuilds    prometheus.Gauge
	HealthCheck     prometheus.Gauge
}

// NewMetrics creates new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		BuildsTotal: *prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "builds_total",
				Help: "Total number of builds processed",
			},
			[]string{"status"},
		),
		BuildDuration: *prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "build_duration_seconds",
				Help: "Build duration in seconds",
			},
			[]string{"project"},
		),
		ActiveBuilds: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_builds",
				Help: "Number of currently active builds",
			},
		),
		HealthCheck: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "health_status",
				Help: "Health status of the service (1 = healthy, 0 = unhealthy)",
			},
		),
	}
}

func (m *Metrics) Register(registry prometheus.Registerer) {
	registry.MustRegister(&m.BuildsTotal)
	registry.MustRegister(&m.BuildDuration)
	registry.MustRegister(m.ActiveBuilds)
	registry.MustRegister(m.HealthCheck)
}

// NewBuildService creates a new build service instance
func NewBuildService(db DatabaseInterface) *BuildService {
	metrics := NewMetrics()
	metrics.Register(prometheus.DefaultRegisterer)
	metrics.HealthCheck.Set(1) // Set initial health status to healthy
	
	return &BuildService{
		db:      db,
		metrics: metrics,
	}
}

// NewBuildServiceWithRegistry creates a new build service instance with custom registry
func NewBuildServiceWithRegistry(db DatabaseInterface, registry prometheus.Registerer) *BuildService {
	metrics := NewMetrics()
	metrics.Register(registry)
	metrics.HealthCheck.Set(1) // Set initial health status to healthy
	
	return &BuildService{
		db:      db,
		metrics: metrics,
	}
}

// Health check endpoint
func (bs *BuildService) healthHandler(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"service":   "build-service",
	}

	// Check database connection
	if err := bs.db.Ping(); err != nil {
		health["status"] = "unhealthy"
		health["database"] = "disconnected"
		bs.metrics.HealthCheck.Set(0)
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		health["database"] = "connected"
		bs.metrics.HealthCheck.Set(1)
		w.WriteHeader(http.StatusOK)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Create build endpoint
func (bs *BuildService) createBuildHandler(w http.ResponseWriter, r *http.Request) {
	var req BuildRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.ProjectName == "" || req.GitURL == "" {
		http.Error(w, "project_name and git_url are required", http.StatusBadRequest)
		return
	}

	if req.Branch == "" {
		req.Branch = "main"
	}

	req.Status = "queued"
	req.CreatedAt = time.Now().UTC()
	req.UpdatedAt = time.Now().UTC()

	// Store in database
	id, err := bs.db.CreateBuild(&req)
	if err != nil {
		log.Printf("Error creating build: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	req.ID = id
	bs.metrics.BuildsTotal.WithLabelValues("queued").Inc()
	bs.metrics.ActiveBuilds.Inc()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)

	// Simulate async build processing
	go bs.processBuild(&req)
}

// Get build endpoint
func (bs *BuildService) getBuildHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr, exists := vars["id"]
	if !exists {
		http.Error(w, "Build ID is required", http.StatusBadRequest)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid build ID", http.StatusBadRequest)
		return
	}

	build, err := bs.db.GetBuild(id)
	if err != nil {
		if err.Error() == "build not found" {
			http.Error(w, "Build not found", http.StatusNotFound)
			return
		}
		log.Printf("Error getting build: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(build)
}

// List builds endpoint
func (bs *BuildService) listBuildsHandler(w http.ResponseWriter, r *http.Request) {
	builds, err := bs.db.ListBuilds()
	if err != nil {
		log.Printf("Error listing builds: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(builds)
}

// Simulate build processing
func (bs *BuildService) processBuild(build *BuildRequest) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		bs.metrics.BuildDuration.WithLabelValues(build.ProjectName).Observe(duration)
		bs.metrics.ActiveBuilds.Dec()
	}()

	// Update status to running
	build.Status = "running"
	build.UpdatedAt = time.Now().UTC()
	if err := bs.db.UpdateBuildStatus(build.ID, "running"); err != nil {
		log.Printf("Error updating build status to running: %v", err)
		return
	}

	// Simulate build time (2-5 seconds)
	time.Sleep(time.Duration(2+len(build.ProjectName)%4) * time.Second)

	// Simulate success/failure (90% success rate)
	success := len(build.ProjectName)%10 != 0

	if success {
		build.Status = "success"
		bs.metrics.BuildsTotal.WithLabelValues("success").Inc()
	} else {
		build.Status = "failed"
		bs.metrics.BuildsTotal.WithLabelValues("failed").Inc()
	}

	build.UpdatedAt = time.Now().UTC()
	if err := bs.db.UpdateBuildStatus(build.ID, build.Status); err != nil {
		log.Printf("Error updating build status to %s: %v", build.Status, err)
	}

	log.Printf("Build %d completed with status: %s", build.ID, build.Status)
}

func main() {
	// Initialize database
	db, err := NewPostgreSQLDatabase()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize tables
	if err := db.InitTables(); err != nil {
		log.Fatalf("Failed to initialize database tables: %v", err)
	}

	// Create build service
	service := NewBuildService(db)

	// Setup router
	router := mux.NewRouter()
	
	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", service.healthHandler).Methods("GET")
	api.HandleFunc("/builds", service.createBuildHandler).Methods("POST")
	api.HandleFunc("/builds", service.listBuildsHandler).Methods("GET")
	api.HandleFunc("/builds/{id}", service.getBuildHandler).Methods("GET")

	// Metrics endpoint
	router.Handle("/metrics", promhttp.Handler())

	// Setup server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting build service on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}