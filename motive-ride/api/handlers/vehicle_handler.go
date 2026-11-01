package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/motive-ride/motive-ride/internal/domain"
	"github.com/motive-ride/motive-ride/internal/service"
)

type VehicleHandler struct {
	svc *service.VehicleService
}

func NewVehicleHandler(svc *service.VehicleService) *VehicleHandler {
	return &VehicleHandler{svc: svc}
}

// POST /api/v1/vehicles
func (h *VehicleHandler) Create(w http.ResponseWriter, r *http.Request) {
	var v domain.Vehicle
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.Create(r.Context(), &v); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

// GET /api/v1/vehicles
func (h *VehicleHandler) List(w http.ResponseWriter, r *http.Request) {
	vehicles, err := h.svc.ListAvailable(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, vehicles)
}

// GET /api/v1/vehicles/{id}
func (h *VehicleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	v, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "vehicle not found")
		return
	}
	writeJSON(w, http.StatusOK, v)
}
