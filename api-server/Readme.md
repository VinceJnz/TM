# Info


## sqlx

Illustrated guide to SQLX
<https://jmoiron.github.io/sqlx/>



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
