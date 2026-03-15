package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/jackc/pgx/v5/pgxpool"
)

type bookingCreatedEvent struct {
	Type       string         `json:"type"`
	Version    int            `json:"version"`
	OccurredAt string         `json:"occurred_at"`
	Booking    bookingPayload `json:"booking"`
}

type bookingPayload struct {
	ID              string  `json:"id"`
	UserID          string  `json:"user_id"`
	HotelID         string  `json:"hotel_id"`
	PromoCode       string  `json:"promo_code"`
	DiscountPercent float64 `json:"discount_percent"`
	Price           float64 `json:"price"`
	CreatedAt       string  `json:"created_at"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	db, err := pgxpool.New(ctx, buildDSN())
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer db.Close()

	brokers := splitComma(getenv("KAFKA_BROKER", "kafka:9092"))
	topic := getenv("KAFKA_TOPIC", "booking-events")
	group := getenv("KAFKA_GROUP", "booking-history-service")

	cfg := sarama.NewConfig()
	cfg.Consumer.Offsets.Initial = sarama.OffsetOldest

	cg, err := sarama.NewConsumerGroup(brokers, group, cfg)
	if err != nil {
		log.Fatalf("kafka consumer group init failed: %v", err)
	}
	defer cg.Close()

	handler := &bookingEventsHandler{db: db, topic: topic}

	for ctx.Err() == nil {
		if err := cg.Consume(ctx, []string{topic}, handler); err != nil {
			// Consume returns on rebalances too; log and retry unless we are shutting down.
			if ctx.Err() != nil {
				break
			}
			log.Printf("kafka consume error: %v", err)
			time.Sleep(500 * time.Millisecond)
		}
	}
}

type bookingEventsHandler struct {
	db    *pgxpool.Pool
	topic string
}

func (h *bookingEventsHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *bookingEventsHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }

func (h *bookingEventsHandler) ConsumeClaim(sess sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		log.Printf("message topic=%s partition=%d offset=%d", msg.Topic, msg.Partition, msg.Offset)
		if err := h.handleMessage(sess.Context(), msg); err != nil {
			// Skip poison messages but keep the consumer running.
			log.Printf("message handling failed topic=%s partition=%d offset=%d err=%v", msg.Topic, msg.Partition, msg.Offset, err)
		} else {
			sess.MarkMessage(msg, "")
		}
	}
	return nil
}

func (h *bookingEventsHandler) handleMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	var ev bookingCreatedEvent
	if err := json.Unmarshal(msg.Value, &ev); err != nil {
		return fmt.Errorf("json unmarshal failed: %w", err)
	}
	if ev.Type != "booking.created" {
		return nil
	}

	createdAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(ev.Booking.CreatedAt))
	if err != nil {
		// booking-service repo uses RFC3339; accept that too.
		createdAt, err = time.Parse(time.RFC3339, strings.TrimSpace(ev.Booking.CreatedAt))
		if err != nil {
			return fmt.Errorf("created_at parse failed: %w", err)
		}
	}

	// Insert booking row without `id`: in the existing schema it can be GENERATED ALWAYS AS IDENTITY.
	_, err = h.db.Exec(ctx, `
INSERT INTO bookings (user_id, hotel_id, promo_code, discount_percent, price, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
`,
		ev.Booking.UserID,
		ev.Booking.HotelID,
		nullableString(ev.Booking.PromoCode),
		ev.Booking.DiscountPercent,
		ev.Booking.Price,
		createdAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("db insert failed: %w", err)
	}

	return nil
}

func nullableString(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func buildDSN() string {
	return "postgres://hotelio:hotelio@10.0.3.15:5433/hotelio?sslmode=disable"
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
