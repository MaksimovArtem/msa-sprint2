package model

import "time"

type Booking struct {
	ID        string
	UserID    string
	HotelID   string
	DateFrom  time.Time
	DateTo    time.Time
	CreatedAt time.Time
}