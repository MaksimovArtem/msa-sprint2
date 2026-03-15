package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"booking-service/internal/kafka"
	bookingpb "booking-service/internal/proto"
	"booking-service/internal/repository"
	"booking-service/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

// bookingServer проксирует gRPC вызовы к сервису
type bookingServer struct {
	bookingpb.UnimplementedBookingServiceServer
	svc *service.BookingService
}

func (s *bookingServer) CreateBooking(ctx context.Context, req *bookingpb.BookingRequest) (*bookingpb.BookingResponse, error) {
	return s.svc.CreateBooking(ctx, req)
}

func (s *bookingServer) ListBookings(ctx context.Context, req *bookingpb.BookingListRequest) (*bookingpb.BookingListResponse, error) {
	userID := ""
	if req != nil {
		userID = req.GetUserId()
	}
	
	fmt.Println("id:", userID)

	bookings, err := s.svc.ListBookings(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &bookingpb.BookingListResponse{Bookings: bookings}, nil
}

func main() {
	dsn := "postgres://hotelio:hotelio@10.0.3.15:5433/hotelio?sslmode=disable"

	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("не удалось подключиться к БД: %v", err)
	}
	defer db.Close()

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("не удалось слушать порт 9090: %v", err)
	}

	repo := repository.NewBookingRepository(db) // <- пул в репозиторий
	svc := service.NewBookingService(repo)      // <- репозиторий в сервис

	// Kafka producer (optional but enabled by default when KAFKA_BROKER is set).
	kafkaBroker := getenv("KAFKA_BROKER", "")
	if kafkaBroker != "" {
		brokers := splitComma(kafkaBroker)
		topic := getenv("KAFKA_TOPIC", "booking-events")
		prod, err := kafka.NewProducerWithTopic(brokers, topic)
		if err != nil {
			log.Fatalf("не удалось создать Kafka producer: %v", err)
		}
		defer prod.Close()
		svc.SetKafkaProducer(prod)
	}

	grpcServer := grpc.NewServer()
	bookingpb.RegisterBookingServiceServer(grpcServer, &bookingServer{svc: svc})
	fmt.Println("Booking gRPC сервис запущен на :9090")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("не удалось запустить сервер: %v", err)
	}
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
