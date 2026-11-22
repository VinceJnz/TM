# TM (Trip Manager) Requirements

## Trips

The original concept for trips was a sheet of paper that people were able to put their name on to book on to a trip. So this UI needs to reflect that level of simplicity, but deal with all the details in the background.

This app/site is about a club that runs trips. It publishes a list of trips (Trip list).

There a 2 fundamental requirements
1. The club organises and publishes trips that members and non-member can participate in by booking a place on the trip.
2. The club has resources such as a lodge, hut and equiment the members can book for their own trips.


There are a number of overlapping considerations.

### Organised club trips:
- A trip leader, or a generic trip contact such as one of the club committiees.
- A maximum capacity
- A location/accomodation that is being used as a base for the trip
- A cost per person (can vary based on age or season e.g. winter/summer rates)
- Can be a one off trip or a reoccuring series of individual trips
- Some trips require that only members are allowed to book, but non-members can participate
- There are different types of trips with different trip difficulty level classifications.
- People need to filter on trip dates, trip type, trip difficulty etc.
- The trip list is used by people wanting to go on a trip. Some of the things they consider, is trip cost, who else is going (e.g. friends), timing of the trip.
- People can book a place on a trip, but their place on the trip is not confirmed until they pay.
- when club Accomodation is used for a trip, the trip essentially owns the Accomodation for the period of the trip. So anyone that wants to go to the Accomodation during that period, must book on the trip.
- Club trips can be catered, people will be asked to advise any dietary requirements.
- People on the trip can be asked if they can provide/share vehicles.

### Member trips that use club resources
- A member can book one or more resources for use on their own trips as long as it is not being used for a club trip.

### Club Recources/Facilities
- The club has resources such as equipment (tents, PLBs, etc), transport (van), and accomodation (Lodge, Hut)
- Resource bookings need to be managed so that a resource cannot be double booked.
- Some resources incure charges that the member must pay at the time of booking.
- Booking fees can vary by season, age, membership status, and potententailly other reasons.
- Accomodation can be booked for exclusive use by a group of people. There is special rate for this. Sometimes the rate is negoiated.
- A booking can be cancelled, and this might incure a penality fee.
- Resources have a maxium capacity, generally 1 for equipment, and accommodation could be many with minimum to avoid inefficient economic use of the resource.


### Booking process

People making bookings need to be able to step through a booking process (move backwards and forwards through the process)

It needs to be possible to go through all these stages in sequence (forwards/backwards). At the moment only setp 1 can be done.

The trip booking process has several stages.
1. Make a booking: The user chooses the start and end date, adds notes, and a voucher code
2. Add people to the booking
3. Review the booking
4. Pay for the booking
5. There may be occasion to cancel a booking. This may incure a penality depending on the timing of the cancellation.

### Future considerations
There needs to be a resource (lodge or hut) booking process. E.g. if an individual wants to run their own trip and use a club resource.
There could be several individuals as a group or independantly making such bookings.

At some point in the future it may be worth being able to book other club resources such a tents, PLB's, etc. This is a future consideration and should not hold up the booking capability.

The club may want to give vouchers to member, non-members etc, for special pricing.

There may be other requirements that I have missed.