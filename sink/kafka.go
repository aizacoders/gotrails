package sink

import (
	"context"
	"time"
)

// KafkaProducer is an interface for producing messages to Kafka
// Replace this with your actual Kafka producer interface if needed
type KafkaProducer interface {
	Produce(ctx context.Context, topic string, key, value []byte) error
}

// IntegrationKafkaProducer wraps a KafkaProducer to capture integration events
type IntegrationKafkaProducer struct {
	Base KafkaProducer
}

func (p *IntegrationKafkaProducer) Produce(ctx context.Context, topic string, key, value []byte) error {
	start := time.Now()
	err := p.Base.Produce(ctx, topic, key, value)
	latency := time.Since(start)

	integration := map[string]any{
		"type":    "kafka",
		"topic":   topic,
		"latency": latency,
		"error":   err,
	}

	// Attach integration to trail in context if present
	trail := ctx.Value("gotrails_trail")
	if trail != nil {
		if t, ok := trail.(interface{ AddIntegration(any) }); ok {
			t.AddIntegration(integration)
		}
	}

	return err
}

// NewIntegrationKafkaProducer wraps a KafkaProducer
func NewIntegrationKafkaProducer(base KafkaProducer) KafkaProducer {
	return &IntegrationKafkaProducer{Base: base}
}
