package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	bookingpb "booking-service/internal/proto"
	"booking-service/internal/repository"
)

// BookingService управляет бизнес-логикой
type BookingService struct {
	repo *repository.BookingRepository
}

func NewBookingService(repo *repository.BookingRepository) *BookingService {
	return &BookingService{repo: repo}
}

// // CreateBooking выполняет логику создания брони
// func (s *BookingService) CreateBooking(ctx context.Context, req *bookingpb.BookingRequest) (*bookingpb.BookingResponse, error) {
// 	// Тут можно добавить проверку промо-кода, скидок, доступности отеля и т.д.
// 	return s.repo.InsertBooking(ctx, req)
// }

const monolithBaseURL = "http://172.18.104.98:8084"

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
	return s.repo.InsertBooking(ctx, req)
}

// --- REST проверки ---

func doGetBool(url string) (bool, error) {
	fmt.Println("url :", url)
	client := http.Client{Timeout: 5 * time.Second, Transport: &http.Transport{ Proxy: nil }}
	resp, err := client.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("неверный статус %d", resp.StatusCode)
	}

	var result struct {
		Value bool `json:"value"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}
	return result.Value, nil
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