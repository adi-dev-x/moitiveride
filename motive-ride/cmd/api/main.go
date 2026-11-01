package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/motive-ride/motive-ride/api/handlers"
	"github.com/motive-ride/motive-ride/api/middleware"
	"github.com/motive-ride/motive-ride/config"
	"github.com/motive-ride/motive-ride/internal/cache"
	"github.com/motive-ride/motive-ride/internal/maps"
	"github.com/motive-ride/motive-ride/internal/repository"
	"github.com/motive-ride/motive-ride/internal/routing"
	"github.com/motive-ride/motive-ride/internal/service"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// ── Database ──────────────────────────────────────────────────────────
	db, err := repository.NewDB(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	slog.Info("database connected")

	// ── Redis cache ───────────────────────────────────────────────────────
	redisClient := cache.NewClient(cfg.RedisAddr, cfg.RedisPassword)
	if err := redisClient.Ping(ctx); err != nil {
		// Redis is optional — warn but don't abort. The Google Maps provider
		// will still work; it just won't cache responses.
		slog.Warn("redis unavailable, caching disabled", "error", err)
	} else {
		slog.Info("redis connected")
		defer redisClient.Close()
	}

	// ── Traffic / time-matrix provider ───────────────────────────────────
	// If a Google Maps API key is present → use live traffic data with Redis cache.
	// Otherwise → fall back to offline Haversine + Bangalore traffic factors.
	var trafficProvider routing.TimeMatrixProvider
	if cfg.GoogleMapsAPIKey != "" {
		trafficProvider = maps.NewGoogleProvider(cfg.GoogleMapsAPIKey, redisClient)
		slog.Info("traffic provider: google maps distance matrix API")
	} else {
		trafficProvider = routing.NewOfflineProvider()
		slog.Warn("traffic provider: offline (set GOOGLE_MAPS_API_KEY for live traffic)")
	}

	// ── Repositories ──────────────────────────────────────────────────────
	empRepo := repository.NewEmployeeRepository(db)
	vehRepo := repository.NewVehicleRepository(db)
	rteRepo := repository.NewRouteRepository(db)

	// ── Routing engine ────────────────────────────────────────────────────
	engine := routing.NewEngine(cfg.DepotLat, cfg.DepotLng, trafficProvider)

	// ── Services ──────────────────────────────────────────────────────────
	empSvc := service.NewEmployeeService(empRepo)
	vehSvc := service.NewVehicleService(vehRepo)
	rteSvc := service.NewRoutingService(empRepo, vehRepo, rteRepo, engine)

	// ── Handlers ──────────────────────────────────────────────────────────
	empH := handlers.NewEmployeeHandler(empSvc)
	vehH := handlers.NewVehicleHandler(vehSvc)
	rteH := handlers.NewRouteHandler(rteSvc)

	// ── Router ────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`)) //nolint:errcheck
	})

	mux.HandleFunc("POST /api/v1/employees", empH.Create)
	mux.HandleFunc("POST /api/v1/employees/bulk", empH.BulkImport)
	mux.HandleFunc("GET /api/v1/employees/{id}", empH.GetByID)

	mux.HandleFunc("POST /api/v1/vehicles", vehH.Create)
	mux.HandleFunc("GET /api/v1/vehicles", vehH.List)
	mux.HandleFunc("GET /api/v1/vehicles/{id}", vehH.GetByID)

	mux.HandleFunc("POST /api/v1/routes/generate", rteH.Generate)
	mux.HandleFunc("GET /api/v1/routes/{id}", rteH.GetByID)

	// ── HTTP server ───────────────────────────────────────────────────────
	handler := middleware.Recovery(middleware.Logger(mux))
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down")
	shutCtx, shutCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutCancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}
