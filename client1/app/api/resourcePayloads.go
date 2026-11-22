package api

import "time"

type bookingDetailsWrite struct {
	OwnerID     int       `json:"owner_id"`
	TripID      int       `json:"trip_id"`
	Notes       string    `json:"notes"`
	FromDate    time.Time `json:"from_date"`
	ToDate      time.Time `json:"to_date"`
	VoucherCode string    `json:"voucher_code"`
}

func BookingDetailsPayload(item BookingDTO) bookingDetailsWrite {
	return bookingDetailsWrite{OwnerID: item.OwnerID, TripID: item.TripID, Notes: item.Notes, FromDate: item.FromDate, ToDate: item.ToDate, VoucherCode: item.VoucherCode}
}

func MyBookingDetailsPayload(item MyBookingDTO) bookingDetailsWrite {
	return bookingDetailsWrite{OwnerID: item.OwnerID, TripID: item.TripID, Notes: item.Notes, FromDate: item.FromDate, ToDate: item.ToDate, VoucherCode: item.VoucherCode}
}

func BookingPayload(item BookingDTO) BookingWrite {
	return BookingWrite{OwnerID: item.OwnerID, TripID: item.TripID, Notes: item.Notes, FromDate: item.FromDate, ToDate: item.ToDate, BookingPrice: item.BookingPrice}
}
func MyBookingPayload(item MyBookingDTO) BookingWrite {
	return BookingWrite{OwnerID: item.OwnerID, TripID: item.TripID, Notes: item.Notes, FromDate: item.FromDate, ToDate: item.ToDate, BookingPrice: item.BookingPrice}
}
func TripPayload(item TripDTO) TripWrite {
	return TripWrite{Version: item.Version, Name: item.Name, Location: item.Location, DifficultyID: item.DifficultyID, FromDate: item.FromDate, ToDate: item.ToDate, MaxParticipants: item.MaxParticipants, TripStatusID: item.TripStatusID, TripTypeID: item.TripTypeID, TripCostGroupID: item.TripCostGroupID}
}
