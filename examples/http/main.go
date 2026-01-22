package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/aizacoders/gotrails/async"
	"github.com/aizacoders/gotrails/gotrails"
	"github.com/aizacoders/gotrails/middleware"
	"github.com/aizacoders/gotrails/sink"
)

func main() {
	// Create configuration
	cfg := gotrails.NewConfig(
		gotrails.WithServiceName("order-service"),
		gotrails.WithEnvironment("development"),
	)

	// Create stdout sink with pretty print
	stdoutSink := sink.NewStdoutSink(
		sink.WithPrettyPrint(true),
	)

	// Wrap with async sink
	asyncSink := async.NewAsyncSink(stdoutSink, cfg.AsyncQueueSize,
		async.WithWorkers(2),
	)
	defer asyncSink.Close()

	// Create native http middleware
	trailMiddleware := middleware.NewHTTPMiddleware(
		middleware.WithHTTPConfig(cfg),
		middleware.WithHTTPSink(asyncSink),
	)

	// Create native Go ServeMux (Go 1.22+ enhanced routing)
	mux := http.NewServeMux()

	// Define routes with middleware wrapper
	mux.Handle("GET /health", trailMiddleware.HandlerFunc(healthHandler))
	mux.Handle("POST /v1/orders", trailMiddleware.HandlerFunc(createOrderHandler))
	mux.Handle("GET /v1/orders/{id}", trailMiddleware.HandlerFunc(getOrderHandler))

	// Start server
	log.Println("Server starting on :8081")
	log.Fatal(http.ListenAndServe(":8081", mux))
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
