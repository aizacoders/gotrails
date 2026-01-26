package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aizacoders/gotrails/gotrails"
	"github.com/aizacoders/gotrails/middleware"
	"github.com/aizacoders/gotrails/sink"
	"github.com/aizacoders/gotrails/transport"
	"github.com/gin-gonic/gin"
)

func main() {
	// Create configuration
	cfg := gotrails.NewConfig(
		gotrails.WithServiceName("payment-service"),
		gotrails.WithEnvironment("development"),
		gotrails.WithMaskFields([]string{
			"password",
			"token",
			"secret",
			"credit_card",
			"cvv",
		}),
	)

	// Create stdout sink with pretty print for development (no async for debugging)
	stdoutSink := sink.NewStdoutSink(
		sink.WithPrettyPrint(false),
	)

	// Create a Gin router
	r := gin.New()
	r.Use(gin.Recovery())

	// Add gotrails middleware
	// Store trail to your DB after each request.
	r.Use(TrailStoreMiddleware())
	trailMiddleware := middleware.NewGinMiddleware(
		middleware.WithGinConfig(cfg),
		middleware.WithGinSink(stdoutSink),
	)
	r.Use(trailMiddleware.Handler())

	// Define routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
		})
	})

	// Remove all trail logic from handler, handler only business logic
	r.POST("/v1/payments", func(c *gin.Context) {
		var req struct {
			Amount        float64 `json:"amount"`
			PaymentMethod string  `json:"payment_method"`
			CreditCard    string  `json:"credit_card"`
			CVV           string  `json:"cvv"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": err.Error(),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"payment_id": "pay-789",
			"status":     "PENDING",
			"amount":     req.Amount,
		})
	})

	r.GET("/v1/payments/:id", func(c *gin.Context) {
		paymentID := c.Param("id")

		c.JSON(http.StatusOK, gin.H{
			"payment_id": paymentID,
			"status":     "COMPLETED",
			"amount":     150000,
		})
	})

	// Fake external provider endpoint (for integration demo)
	external := gin.New()
	external.POST("/external/charge", func(c *gin.Context) {
		time.Sleep(120 * time.Millisecond)
		c.JSON(http.StatusAccepted, gin.H{
			"provider_id": "ext-789",
			"status":      "PENDING",
		})
	})

	go func() {
		_ = external.Run(":8083")
	}()

	// Simulate external/third-party call to test integration capture
	r.POST("/v1/payments/:id/charge", func(c *gin.Context) {
		paymentID := c.Param("id")

		// Simulated external request/response
		reqPayload := map[string]any{
			"payment_id": paymentID,
			"amount":     150000,
			"currency":   "IDR",
		}

		client := &http.Client{
			Transport: transport.NewHTTPRoundTripper(nil),
			Timeout:   2 * time.Second,
		}

		bodyBytes, err := json.Marshal(reqPayload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		req, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, "http://localhost:8083/external/charge", bytes.NewReader(bodyBytes))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		c.JSON(http.StatusAccepted, gin.H{
			"payment_id": paymentID,
			"status":     "CHARGE_PENDING",
		})
	})

	// Start server
	r.Run(":8082")
}

// TrailStoreMiddleware stores trail data after the request finishes.
func TrailStoreMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		trail := gotrails.GetTrail(c.Request.Context())
		if trail == nil {
			return
		}
		fmt.Println("Storing trail:", trail.TraceID)
		// TODO: save trail to your DB or queue here.
		_ = trail
	}

}
