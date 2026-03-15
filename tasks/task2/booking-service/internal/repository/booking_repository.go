package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	bookingpb "booking-service/internal/proto"

	"github.com/jackc/pgx/v5/pgxpool"
)

// BookingRepository отвечает за работу с БД
type BookingRepository struct {
	db *pgxpool.Pool
}

func NewBookingRepository(db *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{db: db}
}

// InsertBooking сохраняет бронь и возвращает её ID
func (r *BookingRepository) InsertBooking(ctx context.Context, b *bookingpb.BookingRequest) (*bookingpb.BookingResponse, error) {
	createdAt := time.Now().UTC()
	var id string

	// err := r.db.QueryRow(ctx,
	// 	`INSERT INTO bookings (user_id, hotel_id, promo_code, discount_percent, price, created_at)
	// 	 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
	// 	b.GetUserId(), b.GetHotelId(), b.GetPromoCode(), 10.0, 100.0, createdAt,
	// ).Scan(&id)

	// if err != nil {
	// 	return nil, fmt.Errorf("failed to insert booking: %w", err)
	// }

	return &bookingpb.BookingResponse{
		Id:              id,
		UserId:          b.GetUserId(),
		HotelId:         b.GetHotelId(),
		PromoCode:       b.GetPromoCode(),
		DiscountPercent: 10.0,
		Price:           100.0,
		CreatedAt:       createdAt.Format(time.RFC3339),
	}, nil
}

// GetAllBookings возвращает все брони (можно фильтровать по userId)
// func (r *BookingRepository) GetAllBookings(ctx context.Context, userId string) ([]*bookingpb .BookingResponse, error) {
// 	query := `SELECT id, user_id, hotel_id, promo_code, discount_percent, price, created_at FROM bookings`
// 	args := []interface{}{}
// 	if userId != "" {
// 		query += " WHERE user_id=$1"
// 		args = append(args, userId)
// 	}

// 	rows, err := r.db.Query(ctx, query, args...)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to query bookings: %w", err)
// 	}
// 	defer rows.Close()

// 	var bookings []*bookingpb .BookingResponse
// 	for rows.Next() {
// 		var b bookingpb .BookingResponse
// 		var createdAt time.Time
// 		if err := rows.Scan(&b.Id, &b.UserId, &b.HotelId, &b.PromoCode, &b.DiscountPercent, &b.Price, &createdAt); err != nil {
// 			return nil, err
// 		}
// 		b.CreatedAt = createdAt.Format(time.RFC3339)
// 		bookings = append(bookings, &b)
// 	}
// 	return bookings, nil
// }

func (r *BookingRepository) GetAllBookings(ctx context.Context, userId string) ([]*bookingpb.BookingResponse, error) {
	// log.Fatalf("userid: %v", userId)

	var query string
	var args []interface{}

	if userId != "" {
		query = `SELECT id, user_id, hotel_id, promo_code, discount_percent, price, created_at 
		         FROM bookings WHERE user_id=$1`
		args = append(args, userId)
	} else {
		query = `SELECT id, user_id, hotel_id, promo_code, discount_percent, price, created_at FROM bookings`
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query bookings: %w", err)
	}
	defer rows.Close()

	var bookings []*bookingpb.BookingResponse
	for rows.Next() {
		var b bookingpb.BookingResponse
		var id any
		var promoCode sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&id, &b.UserId, &b.HotelId, &promoCode, &b.DiscountPercent, &b.Price, &createdAt); err != nil {
			return nil, err
		}
		b.Id = formatID(id)
		if promoCode.Valid {
			b.PromoCode = promoCode.String
		}
		b.CreatedAt = createdAt.Format(time.RFC3339)
		bookings = append(bookings, &b)
	}

	return bookings, nil
}

func formatID(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []byte:
		return string(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case int32:
		return strconv.FormatInt(int64(t), 10)
	case int:
		return strconv.Itoa(t)
	default:
		return fmt.Sprint(v)
	}
}
