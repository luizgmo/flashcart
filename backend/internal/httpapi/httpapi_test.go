package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"flashcart/backend/internal/config"
)

func TestPaginationBounds(t *testing.T) {
	cases := []struct {
		query      string
		page, size int
		wantErr    bool
	}{
		{"", 1, 12, false}, {"?page=2&page_size=20", 2, 20, false}, {"?page=0", 0, 0, true}, {"?page=-1", 0, 0, true}, {"?page_size=25", 0, 0, true}, {"?page_size=nope", 0, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.query, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/v1/products"+tc.query, nil)
			page, size, err := pagination(r)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && (page != tc.page || size != tc.size) {
				t.Fatalf("got %d/%d want %d/%d", page, size, tc.page, tc.size)
			}
		})
	}
}

func TestValidOriginRequiresExactConfiguredOrigin(t *testing.T) {
	for _, tc := range []struct {
		origin, want string
		ok           bool
	}{{"http://localhost:5173", "http://localhost:5173", true}, {"http://localhost:5173.evil.test", "http://localhost:5173", false}, {"null", "http://localhost:5173", false}, {"", "http://localhost:5173", false}, {"http://localhost:5173", "", false}} {
		r := httptest.NewRequest("POST", "/", nil)
		r.Header.Set("Origin", tc.origin)
		if got := validOrigin(r, tc.want); got != tc.ok {
			t.Fatalf("origin=%q got %v want %v", tc.origin, got, tc.ok)
		}
	}
}

func TestHealthIncludesSecurityHeaders(t *testing.T) {
	cfg := testConfig()
	rec := httptest.NewRecorder()
	New(nil, cfg).ServeHTTP(rec, httptest.NewRequest("GET", "/healthz", nil))
	if rec.Code != 200 {
		t.Fatalf("status=%d", rec.Code)
	}
	for _, header := range []string{"X-Request-ID", "X-Content-Type-Options", "X-Frame-Options", "Content-Security-Policy"} {
		if rec.Header().Get(header) == "" {
			t.Errorf("missing %s", header)
		}
	}
}

func testConfig() config.Config {
	return config.Config{AppEnv: "test", AppOrigin: "http://localhost:5173", AssistantMode: "template", FrontendDist: "../../../frontend/dist"}
}

func TestTemplateAnswerDoesNotOfferPaymentAfterExpiry(t *testing.T) {
	s := assistantSnapshot{Status: "PENDING_PAYMENT", TotalCents: 2500, Currency: "BRL", ExpiresAt: time.Date(2026, 11, 11, 12, 5, 0, 0, time.UTC), CanPay: false}
	answer := templateAnswer(s, "posso pagar?")
	if answer != "O prazo da reserva terminou. A liberação do estoque está em processamento." {
		t.Fatalf("unexpected template answer: %s", answer)
	}
}

func TestGeminiAdapterUsesScopedSnapshotAndValidatesStructuredResponse(t *testing.T) {
	snapshot := assistantSnapshot{OrderID: "4c52ef0d-9c7d-4d84-a4a5-8bff0b887f13", Status: "PAID", TotalCents: 2500, Currency: "BRL", ExpiresAt: time.Now(), ObservedAt: time.Now()}
	rt := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		if !strings.HasPrefix(r.URL.Path, "/v1beta/models/") {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"{\"answer\":\"Pedido pago.\",\"order_id\":\"4c52ef0d-9c7d-4d84-a4a5-8bff0b887f13\",\"status\":\"PAID\"}"}]}}]}`))}, nil
	})
	answer, err := askGeminiWithClient(context.Background(), config.Config{GeminiAPIKey: "test-key", GeminiModel: "test-model", GeminiAPIEndpoint: "https://model.test"}, snapshot, "Qual o status?", &http.Client{Transport: rt})
	if err != nil {
		t.Fatal(err)
	}
	if answer != "Pedido pago." {
		t.Fatalf("answer=%q", answer)
	}
}

func TestGeminiAdapterRejectsResponseForAnotherOrder(t *testing.T) {
	snapshot := assistantSnapshot{OrderID: "4c52ef0d-9c7d-4d84-a4a5-8bff0b887f13", Status: "PAID"}
	rt := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"{\"answer\":\"Pedido pago.\",\"order_id\":\"00000000-0000-4000-8000-000000000000\",\"status\":\"PAID\"}"}]}}]}`))}, nil
	})
	_, err := askGeminiWithClient(context.Background(), config.Config{GeminiAPIKey: "test", GeminiModel: "test", GeminiAPIEndpoint: "https://model.test"}, snapshot, "status?", &http.Client{Transport: rt})
	if err == nil || err.Error() != "invalid_response" {
		t.Fatalf("error=%v, want invalid_response", err)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
