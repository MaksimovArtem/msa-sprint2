package main

import (
	"context"
	"fmt"
	"log"
	"net"

	bookingpb "booking-service/internal/proto"
	"booking-service/internal/repository"
	"booking-service/internal/service"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc"
)

// bookingServer проксирует gRPC вызовы к сервису
type bookingServer struct {
	bookingpb .UnimplementedBookingServiceServer
	svc *service.BookingService
}

func (s *bookingServer) CreateBooking(ctx context.Context, req *bookingpb .BookingRequest) (*bookingpb .BookingResponse, error) {
	return s.svc.CreateBooking(ctx, req)
}

func (s *bookingServer) ListBookings(ctx context.Context, req *bookingpb .BookingListRequest) (*bookingpb .BookingListResponse, error) {
	bookings, err := s.svc.ListBookings(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &bookingpb .BookingListResponse{Bookings: bookings}, nil
}

func main() {
	dsn := "postgres://hotelio:hotelio@172.18.104.98:5433/hotelio?sslmode=disable"
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("не удалось подключиться к БД: %v", err)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Fatalf("не удалось пинговать БД: %v", err)
	}

	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatalf("не удалось слушать порт 9090: %v", err)
	}

	repo := repository.NewBookingRepository(db) // <- пул в репозиторий
	svc := service.NewBookingService(repo)      // <- репозиторий в сервис
	grpcServer := grpc.NewServer()
	bookingpb .RegisterBookingServiceServer(grpcServer, &bookingServer{svc: svc})
	fmt.Println("Booking gRPC сервис запущен на :9090")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("не удалось запустить сервер: %v", err)
	}
}