package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRespondJSON_Success(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]string{"message": "success"}
	respondJSON(rr, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, rr.Body.String(), "success")
}

func TestRespondJSON_Created(t *testing.T) {
	rr := httptest.NewRecorder()

	data := map[string]int{"id": 123}
	respondJSON(rr, http.StatusCreated, data)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.Contains(t, rr.Body.String(), "123")
}

func TestRespondJSON_EmptyData(t *testing.T) {
	rr := httptest.NewRecorder()

	respondJSON(rr, http.StatusOK, nil)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Body.String()) // nil data results in no body
}

func TestRespondJSON_Array(t *testing.T) {
	rr := httptest.NewRecorder()

	data := []string{"a", "b", "c"}
	respondJSON(rr, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), `["a","b","c"]`)
}

func TestRespondError_BadRequest(t *testing.T) {
	rr := httptest.NewRecorder()

	respondError(rr, http.StatusBadRequest, "invalid input")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid input")
}

func TestRespondError_Unauthorized(t *testing.T) {
	rr := httptest.NewRecorder()

	respondError(rr, http.StatusUnauthorized, "not authorized")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "not authorized")
}

func TestRespondError_NotFound(t *testing.T) {
	rr := httptest.NewRecorder()

	respondError(rr, http.StatusNotFound, "resource not found")

	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Contains(t, rr.Body.String(), "resource not found")
}

func TestRespondError_InternalServerError(t *testing.T) {
	rr := httptest.NewRecorder()

	respondError(rr, http.StatusInternalServerError, "internal error")

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "internal error")
}

func TestRespondError_Conflict(t *testing.T) {
	rr := httptest.NewRecorder()

	respondError(rr, http.StatusConflict, "resource already exists")

	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "resource already exists")
}
