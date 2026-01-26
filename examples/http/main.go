package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"context"

	"github.com/aizacoders/gotrails/gotrails"
	"github.com/aizacoders/gotrails/middleware"
	"github.com/aizacoders/gotrails/sink"
	"github.com/aizacoders/gotrails/transport"
)

func main() {
	// Create configuration
	cfg := gotrails.NewConfig(
		gotrails.WithServiceName("order-service"),
		gotrails.WithEnvironment("development"),
	)

	// Create stdout sink with pretty print
	stdoutSink := sink.NewStdoutSink(
		sink.WithPrettyPrint(false),
	)

	// Create native Go ServeMux (Go 1.22+ enhanced routing)
	mux := http.NewServeMux()

	// Define routes with middleware wrapper
	mux.Handle("GET /health", http.HandlerFunc(healthHandler))
	mux.Handle("POST /v1/orders", http.HandlerFunc(createOrderHandler))
	mux.Handle("GET /v1/orders/{id}", http.HandlerFunc(getOrderHandler))
	mux.Handle("POST /v1/orders/{id}/charge", http.HandlerFunc(chargeOrderHandler))

	go startExternalServer()

	// Start server
	log.Println("Server starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", buildHandler(cfg, stdoutSink, mux)))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	// Get trail from context
	ctx := r.Context()
	if trail := gotrails.GetTrail(ctx); trail != nil {
		trail.SetMetadata("user_id", "u-456")
	}

	// Parse request
	var req struct {
		Items []struct {
			ProductID string  `json:"product_id"`
			Quantity  int     `json:"quantity"`
			Price     float64 `json:"price"`
		} `json:"items"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": err.Error(),
		})
		return
	}

	// Success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"order_id": "ord-123",
		"status":   "CREATED",
		"items":    len(req.Items),
	})
}

func getOrderHandler(w http.ResponseWriter, r *http.Request) {
	// Get path parameter using Go 1.22+ PathValue
	orderID := r.PathValue("id")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"order_id": orderID,
		"status":   "PROCESSING",
		"total":    250000,
	})
}

func chargeOrderHandler(w http.ResponseWriter, r *http.Request) {
	orderID := r.PathValue("id")

	reqPayload := map[string]any{
		"order_id": orderID,
		"amount":   250000,
		"currency": "IDR",
	}

	bodyBytes, err := json.Marshal(reqPayload)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	client := &http.Client{
		Transport: transport.NewHTTPRoundTripper(nil),
		Timeout:   2 * time.Second,
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, "http://localhost:8083/external/charge", bytes.NewReader(bodyBytes))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"order_id": orderID,
		"status":   "CHARGE_PENDING",
	})
}

func buildHandler(cfg *gotrails.Config, stdoutSink sink.Sink, mux http.Handler) http.Handler {
	trailMiddleware := middleware.NewHTTPMiddleware(
		middleware.WithHTTPConfig(cfg),
		middleware.WithHTTPSink(stdoutSink),
		middleware.WithHTTPAfterFlush(func(ctx context.Context, trail *gotrails.Trail) {
			if trail == nil {
				return
			}
			log.Println("Storing audit trails in DB:", trail.TraceID)
			// TODO: save trail to your DB or queue here.
			_ = ctx
		}),
	)
	return trailMiddleware.Middleware()(mux)
}

func startExternalServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /external/charge", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(120 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"provider_id": "ext-789",
			"status":      "PENDING",
		})
	})
	log.Println("External server starting on :8083")
	_ = http.ListenAndServe(":8083", mux)
}
