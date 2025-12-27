package builder

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseWriter_Toast(t *testing.T) {
	tests := []struct {
		name      string
		toastType ToastType
		message   string
		wantType  string
	}{
		{
			name:      "success toast",
			toastType: ToastSuccess,
			message:   "Operation succeeded",
			wantType:  "success",
		},
		{
			name:      "error toast",
			toastType: ToastError,
			message:   "Operation failed",
			wantType:  "error",
		},
		{
			name:      "info toast",
			toastType: ToastInfo,
			message:   "Information message",
			wantType:  "info",
		},
		{
			name:      "warning toast",
			toastType: ToastWarning,
			message:   "Warning message",
			wantType:  "warning",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := NewResponseWriter(w)

			rw.Toast(tt.toastType, tt.message)

			// Check that HX-Trigger header is set
			trigger := w.Header().Get("HX-Trigger")
			if trigger == "" {
				t.Error("HX-Trigger header not set")
			}

			// Basic validation that it contains the message
			if trigger != "" && len(trigger) < 10 {
				t.Errorf("HX-Trigger header seems invalid: %s", trigger)
			}
		})
	}
}

func TestResponseWriter_Redirect(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	rw.Redirect("/test/path")

	hxRedirect := w.Header().Get("HX-Redirect")
	if hxRedirect != "/test/path" {
		t.Errorf("HX-Redirect = %q, want %q", hxRedirect, "/test/path")
	}
}

func TestResponseWriter_Success(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	rw.Success("Test success")

	trigger := w.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("HX-Trigger header not set for success toast")
	}
}

func TestResponseWriter_Error(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	rw.Error("Test error")

	trigger := w.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("HX-Trigger header not set for error toast")
	}
}

func TestResponseWriter_OKWithToast(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	rw.OKWithToast(ToastSuccess, "Operation complete")

	if w.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
	}

	trigger := w.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("HX-Trigger header not set")
	}
}

func TestResponseWriter_Trigger(t *testing.T) {
	w := httptest.NewRecorder()
	rw := NewResponseWriter(w)

	events := map[string]interface{}{
		"customEvent": "value",
		"anotherEvent": map[string]string{
			"key": "value",
		},
	}

	rw.Trigger(events)

	trigger := w.Header().Get("HX-Trigger")
	if trigger == "" {
		t.Error("HX-Trigger header not set for custom events")
	}
}
