# Info

myuser
mypassword

## Env

<https://towardsdatascience.com/use-environment-variable-in-your-next-golang-project-39e17c3aaa66>


## Cookie handling

there's a whole bunch of settings needed to make cookies work correctly.
see chatGpt thread.

For cookies to be accepted in the browser:
Modern browsers will only accept session cookies via https
The wasm client needs to be served from an https server


## sqlx

Illustrated guide to SQLX
<https://jmoiron.github.io/sqlx/>




## data access management

User groups provide members access to records (aka resources). The records are filtered by ownership.
User groups have an admin flag. If this is set then members of the group can access all the records of the associated resources regardless of ownership.


## New queries

Count the number participants above and below max_participants threshold

```sql
WITH booking_order AS (
    SELECT 
        atb.trip_id,
        atbp.id as participant_id,
        ROW_NUMBER() OVER (PARTITION BY atb.trip_id ORDER BY atbp.id ASC) AS booking_position
    FROM public.at_booking_people atbp
	JOIN public.at_bookings atb ON atb.id=atbp.booking_id
)
SELECT 
    trip_id,
    SUM(CASE WHEN booking_position <= att.max_participants THEN 1 ELSE 0 END) AS before_threshold,
    SUM(CASE WHEN booking_position > att.max_participants THEN 1 ELSE 0 END) AS after_threshold,
	STRING_AGG(CASE WHEN booking_position <= att.max_participants THEN participant_id::text END, ', ') AS before_threshold_ids,
    STRING_AGG(CASE WHEN booking_position > att.max_participants THEN participant_id::text END, ', ') AS after_threshold_ids
FROM booking_order
JOIN public.at_trips att ON att.id=booking_order.trip_id
GROUP BY trip_id;
```

List participants above and below max_participants threshold

```sql
WITH booking_order AS (
    SELECT 
        atb.trip_id,
		atb.id as booking_id,
        atbp.id as participant_id,
        atbp.person_id as person_id,
	    ROW_NUMBER() OVER (PARTITION BY atb.trip_id ORDER BY atbp.id ASC) AS booking_position
    FROM public.at_booking_people atbp
	JOIN public.at_bookings atb ON atb.id=atbp.booking_id
)
SELECT 
    trip_id,
	booking_order.booking_id,
    participant_id,
	booking_order.person_id,
	stu.name,
    CASE 
        WHEN booking_position <= att.max_participants THEN 'before_threshold' 
        ELSE 'after_threshold' 
    END AS booking_status
FROM booking_order
JOIN public.at_trips att ON att.id=booking_order.trip_id
JOIN public.st_users stu ON stu.id=booking_order.person_id
ORDER BY trip_id, booking_position;
```
