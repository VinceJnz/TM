package tripParticipantStatusReport

import "strings"

func tripDates(trip TableData) string {
	if trip.TripFrom.Equal(trip.TripTo) {
		return trip.TripFrom.Format("Mon 2 Jan 2006")
	}
	return trip.TripFrom.Format("Mon 2 Jan 2006") + " – " + trip.TripTo.Format("Mon 2 Jan 2006")
}

func tripMetadata(trip TableData) string {
	var labels []string
	for _, value := range []string{trip.Location, trip.Type, trip.Grade, trip.Status} {
		if value != "" {
			labels = append(labels, value)
		}
	}
	return strings.Join(labels, " · ")
}
