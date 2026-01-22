package main

import (
	"net/http"

	"github.com/aizacoders/gotrails/gotrails"
	"github.com/aizacoders/gotrails/middleware"
	"github.com/aizacoders/gotrails/sink"
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
		sink.WithPrettyPrint(true),
	)

	// Create Gin router
	r := gin.New()
	r.Use(gin.Recovery())

	// Add gotrails middleware
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

	r.POST("/v1/payments", func(c *gin.Context) {
		// Get trail from context for adding metadata
		ctx := c.Request.Context()
		if trail := gotrails.GetTrail(ctx); trail != nil {
			trail.SetMetadata("user_id", "u-123")
			trail.SetMetadata("merchant_id", "m-456")
		}

		// Simulate payment processing
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

		// Success response
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

	// Start server
	r.Run(":8080")
}
