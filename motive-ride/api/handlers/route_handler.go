package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/motive-ride/motive-ride/internal/domain"
	"github.com/motive-ride/motive-ride/internal/service"
)

type RouteHandler struct {
	svc *service.RoutingService
}

func NewRouteHandler(svc *service.RoutingService) *RouteHandler {
	return &RouteHandler{svc: svc}
}

type generateRouteRequest struct {
	CompanyID string `json:"company_id"`
	ShiftID   string `json:"shift_id"`
	RouteDate string `json:"route_date"` // YYYY-MM-DD
	Direction string `json:"direction"`  // PICKUP | DROP
}

// POST /api/v1/routes/generate
func (h *RouteHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req generateRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.CompanyID == "" || req.ShiftID == "" {
		writeError(w, http.StatusBadRequest, "company_id and shift_id are required")
		return
	}

	routeDate, err := time.Parse("2006-01-02", req.RouteDate)
	if err != nil {
		writeError(w, http.StatusBadRequest, "route_date must be YYYY-MM-DD")
		return
	}

	direction := domain.RouteDirection(req.Direction)
	if direction != domain.DirectionPickup && direction != domain.DirectionDrop {
		writeError(w, http.StatusBadRequest, "direction must be PICKUP or DROP")
		return
	}

	result, err := h.svc.GenerateRoutes(r.Context(), service.GenerateRouteRequest{
		CompanyID: req.CompanyID,
		ShiftID:   req.ShiftID,
		RouteDate: routeDate,
		Direction: direction,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"routes_generated": len(result.Routes),
		"errors":           len(result.Errors),
		"duration_ms":      result.DurationMs,
	})
}

// GET /api/v1/routes/{id}
func (h *RouteHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	manifest, err := h.svc.GetRouteManifest(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}
	writeJSON(w, http.StatusOK, manifest)
}
