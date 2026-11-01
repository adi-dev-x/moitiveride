package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/motive-ride/motive-ride/internal/domain"
	"github.com/motive-ride/motive-ride/internal/service"
)

type EmployeeHandler struct {
	svc *service.EmployeeService
}

func NewEmployeeHandler(svc *service.EmployeeService) *EmployeeHandler {
	return &EmployeeHandler{svc: svc}
}

// POST /api/v1/employees
func (h *EmployeeHandler) Create(w http.ResponseWriter, r *http.Request) {
	var emp domain.Employee
	if err := json.NewDecoder(r.Body).Decode(&emp); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.svc.Create(r.Context(), &emp); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, emp)
}

// POST /api/v1/employees/bulk
func (h *EmployeeHandler) BulkImport(w http.ResponseWriter, r *http.Request) {
	var employees []domain.Employee
	if err := json.NewDecoder(r.Body).Decode(&employees); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := h.svc.BulkImport(r.Context(), employees); err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int{"imported": len(employees)})
}

// GET /api/v1/employees/{id}
func (h *EmployeeHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	emp, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "employee not found")
		return
	}
	writeJSON(w, http.StatusOK, emp)
}
