package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"booking-service/internal/kafka"
	bookingpb "booking-service/internal/proto"
	"booking-service/internal/repository"
)

// BookingService управляет бизнес-логикой
type BookingService struct {
	repo      *repository.BookingRepository
	producer  *kafka.Producer
	eventTime func() time.Time
}

func NewBookingService(repo *repository.BookingRepository) *BookingService {
	return &BookingService{
		repo:      repo,
		eventTime: time.Now,
	}
}

func (s *BookingService) SetKafkaProducer(producer *kafka.Producer) {
	s.producer = producer
}

// // CreateBooking выполняет логику создания брони
// func (s *BookingService) CreateBooking(ctx context.Context, req *bookingpb.BookingRequest) (*bookingpb.BookingResponse, error) {
// 	// Тут можно добавить проверку промо-кода, скидок, доступности отеля и т.д.
// 	return s.repo.InsertBooking(ctx, req)
// }

const monolithBaseURL = "http://10.0.3.15:8084"

// CreateBooking выполняет логику создания брони с проверками пользователя и отеля
func (s *BookingService) CreateBooking(ctx context.Context, req *bookingpb.BookingRequest) (*bookingpb.BookingResponse, error) {
	// Проверка пользователя
	if ok, err := checkUserActive(req.UserId); err != nil {
		return nil, fmt.Errorf("ошибка проверки пользователя: %w", err)
	} else if !ok {
		return nil, fmt.Errorf("пользователь неактивен")
	}

	if ok, err := checkUserBlacklisted(req.UserId); err != nil {
		return nil, fmt.Errorf("ошибка проверки черного списка: %w", err)
	} else if ok {
		return nil, fmt.Errorf("пользователь в черном списке")
	}

	// Проверка отеля
	if ok, err := checkHotelTrusted(req.HotelId); err != nil {
		return nil, fmt.Errorf("ошибка проверки отзывов отеля: %w", err)
	} else if !ok {
		return nil, fmt.Errorf("отель ненадежный")
	}

	if ok, err := checkHotelOperational(req.HotelId); err != nil {
		return nil, fmt.Errorf("ошибка проверки работы отеля: %w", err)
	} else if !ok {
		return nil, fmt.Errorf("отель не работает")
	}

	if ok, err := checkHotelFullyBooked(req.HotelId); err != nil {
		return nil, fmt.Errorf("ошибка проверки полной занятости отеля: %w", err)
	} else if ok {
		return nil, fmt.Errorf("отель полностью забронирован")
	}

	// Вставка брони
	booking, err := s.repo.InsertBooking(ctx, req)
	if err != nil {
		return nil, err
	}

	if s.producer != nil {
		event := struct {
			Type       string                     `json:"type"`
			Version    int                        `json:"version"`
			OccurredAt string                     `json:"occurred_at"`
			Booking    *bookingpb.BookingResponse `json:"booking"`
		}{
			Type:       "booking.created",
			Version:    1,
			OccurredAt: s.eventTime().UTC().Format(time.RFC3339Nano),
			Booking:    booking,
		}

		partition, offset, err := s.producer.SendBookingCreated(event)
		if err != nil {
			return nil, fmt.Errorf("не удалось отправить событие в Kafka: %w", err)
		}
		if offset < 0 {
			return nil, fmt.Errorf("Kafka не вернула offset (partition=%d offset=%d)", partition, offset)
		}

		// This is a broker ack (producer is configured with acks=all).
		fmt.Printf("kafka ack: topic=%s partition=%d offset=%d\n", s.producer.Topic(), partition, offset)
	}

	return booking, nil
}

// --- REST проверки ---

func doGetBool(url string) (bool, error) {
	client := http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{Proxy: nil}}
	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("неверный статус %d", resp.StatusCode)
	}

	// Монолит возвращает JSON-скаляр `true/false` (а не объект вида {"value":true}).
	// Чтобы не падать на несовпадении формата, пробуем оба варианта.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var b bool
	if err := json.Unmarshal(body, &b); err == nil {
		fmt.Println("result:", b)
		return b, nil
	}

	var wrapped struct {
		Value bool `json:"value"`
	}
	if err := json.Unmarshal(body, &wrapped); err == nil {
		fmt.Println("result:", wrapped.Value)
		return wrapped.Value, nil
	}

	return false, fmt.Errorf("неожиданный ответ для bool: %s", strings.TrimSpace(string(body)))
}

// Пользователь
func checkUserActive(userId string) (bool, error) {
	return doGetBool(fmt.Sprintf("%s/api/users/%s/active", monolithBaseURL, userId))
}

func checkUserBlacklisted(userId string) (bool, error) {
	return doGetBool(fmt.Sprintf("%s/api/users/%s/blacklisted", monolithBaseURL, userId))
}

// Отель
func checkHotelTrusted(hotelId string) (bool, error) {
	return doGetBool(fmt.Sprintf("%s/api/reviews/hotel/%s/trusted", monolithBaseURL, hotelId))
}

func checkHotelOperational(hotelId string) (bool, error) {
	return doGetBool(fmt.Sprintf("%s/api/hotels/%s/operational", monolithBaseURL, hotelId))
}

func checkHotelFullyBooked(hotelId string) (bool, error) {
	return doGetBool(fmt.Sprintf("%s/api/hotels/%s/fully-booked", monolithBaseURL, hotelId))
}

// ListBookings возвращает список бронирований
func (s *BookingService) ListBookings(ctx context.Context, userId string) ([]*bookingpb.BookingResponse, error) {
	return s.repo.GetAllBookings(ctx, userId)
}
