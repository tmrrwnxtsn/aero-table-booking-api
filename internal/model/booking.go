package model

import "time"

// Booking представляет бронь.
type Booking struct {
	ID             uint64    `json:"id"`
	ClientID       uint64    `json:"client_id"`
	BookedDate     time.Time `json:"booked_date"`
	BookedTimeFrom time.Time `json:"booked_time_from"`
	BookedTimeTo   time.Time `json:"booked_time_to"`
}

// BookingsTables представляет таблицу в БД, в которой хранятся столики и брони, к которым они относятся.
type BookingsTables struct {
	ID        uint64
	BookingID uint64
	TableID   uint64
}
