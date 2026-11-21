package api

func BookingPayload(item BookingDTO) BookingWrite {
	return BookingWrite{OwnerID: item.OwnerID, TripID: item.TripID, Notes: item.Notes, FromDate: item.FromDate, ToDate: item.ToDate, BookingPrice: item.BookingPrice}
}
func MyBookingPayload(item MyBookingDTO) BookingWrite {
	return BookingWrite{OwnerID: item.OwnerID, TripID: item.TripID, Notes: item.Notes, FromDate: item.FromDate, ToDate: item.ToDate, BookingPrice: item.BookingPrice}
}
func TripPayload(item TripDTO) TripWrite {
	return TripWrite{Name: item.Name, Location: item.Location, DifficultyID: item.DifficultyID, FromDate: item.FromDate, ToDate: item.ToDate, MaxParticipants: item.MaxParticipants, TripStatusID: item.TripStatusID, TripTypeID: item.TripTypeID, TripCostGroupID: item.TripCostGroupID}
}
