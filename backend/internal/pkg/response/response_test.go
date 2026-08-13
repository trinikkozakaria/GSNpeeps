package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSuccess(t *testing.T) {
	recorder := httptest.NewRecorder()
	Success(recorder, http.StatusOK, map[string]string{"status": "ok"}, "Berhasil")
	var body Envelope
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || !body.Success {
		t.Fatalf("unexpected response: code=%d body=%+v", recorder.Code, body)
	}
}

func TestPaginated(t *testing.T) {
	recorder := httptest.NewRecorder()
	Paginated(recorder, []string{}, PaginationMeta{Page: 1, Limit: 20, TotalData: 0, TotalPage: 0}, "")
	var body Envelope
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Meta == nil || body.Meta.Limit != 20 {
		t.Fatalf("unexpected meta: %+v", body.Meta)
	}
}

func TestValidationError(t *testing.T) {
	recorder := httptest.NewRecorder()
	ValidationError(recorder, map[string]string{"email": "Email tidak valid"})
	if recorder.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", recorder.Code)
	}
}

func TestEmptySuccessIncludesNullData(t *testing.T) {
	recorder := httptest.NewRecorder()
	EmptySuccess(recorder, "Logout berhasil")

	var body map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if _, exists := body["data"]; !exists || body["data"] != nil {
		t.Fatalf("data = %#v, want explicit null", body["data"])
	}
}
