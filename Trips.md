# Trips

This needs some significant rework.

This app/site is about a club that runs trips. It publishes a list of trips (Trip list).

There are a number of overlapping considerations.

Each trip (organised trip) has:
- A trip leader, or a generic trip contact such as one of the club committiees.
- A maximum capacity
- A location that is being used as a base for the trip
- A cost per person (can vary based on age or season e.g. winter/summer rates)
- Can be a one off trip or a reoccuring series of individual trips
- Some trips require that only members are allowed to book, but non-members can participate
- There are member and non-member rates for some of the facilities used for a trip (e.g. the clubs Ski lodge, or a club hut)
- There are different types of trips with different trip difficulty level classifications.
- People need to filter on trip dates, trip type.
- The trip list is used by people wanting to go on a trip. Some of the things they consider, is trip cost, who else is going (e.g. friends), timing of the trip.
- People can book on a trip, but their place on the trip is not confirmed until they pay.
- it's possible to make a group booking, and this might be done by an external organisation.
- It is possible book a whole club lodge or hut and there is a price that reflects this sort of booking.
- It is possible to make an individual booking for a club facility (lodge or hut) as long as there is not an organised trip using it at the time the individual wants to book.
- when one of the club lodges or huts are involved the trip essentially owns the facility for the period of the trip. So anyone that wants to go to the facility must book on the trip.

The original concept was a sheet of paper that people were able to put their name on to book on to a trip. So this UI needs to reflect that level of simplicity, but deal with all the details in the background.

People making bookings need to be able to step through the booking process (move backwards and forwards through the process)

It needs to be possible to go through all these stages in sequence (forwards/backwards). At the moment only setp 1 can be done.

The trip booking process has several stages.
1. Make a booking: The user chooses the start and end date, adds notes, and a voucher code
2. Add people to the booking
3. Review the booking
4. Pay for the booking
5. There may be occasion to cancel a booking. This may incure a penality depending on the timing of the cancellation.

There also needs to be a facility (lodge or hut) booking process. E.g. if an individul wants to run their own trip and use a club facility.
There could be several individuals as a group or independantly making such bookings.

At some point in the future it may be worth being able to book other club resources such a tents, PLB's, etc. This is a future consideration and should not hold up the booking capability.

There may be other requirements that I have missed.