package bookings

const myListQuery = `SELECT atb.id,atb.owner_id,atb.trip_id,atb.notes,atb.from_date,atb.to_date,atb.booking_status_id,atb.created,atb.modified,atb.booking_date,atb.group_booking_id,atb.booking_price,atb.payment_date,atb.amount_paid,atb.currency,atb.stripe_session_id, atb.voucher_code, EXISTS (SELECT 1 FROM at_payments p WHERE p.booking_id=atb.id) AS has_payment_history,
				CASE WHEN atb.free_confirmed_at IS NOT NULL AND atb.booking_status_id=3 THEN 'Confirmed' ELSE ebs.status END AS status, COUNT(stu.name) as participants,
				SUM(attc.amount) * (EXTRACT(EPOCH FROM (atb.to_date - atb.from_date)) / 86400) as booking_cost,
				--SUM(attc.amount) AS booking_cost,
				att.trip_name, stu2.name as owner_name
				FROM at_trips att
				LEFT JOIN at_bookings atb ON atb.trip_id=att.id
				LEFT JOIN at_booking_people atbp ON atbp.booking_id=atb.id
					 JOIN et_booking_status ebs on ebs.id=atb.booking_status_id
				LEFT JOIN st_users stu ON stu.id=atbp.person_id
				LEFT JOIN at_trip_cost_groups attcg ON attcg.id=att.trip_cost_group_id
				LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id=att.trip_cost_group_id
										AND attc.member_status_id=stu.member_status_id
										AND attc.user_age_group_id=stu.user_age_group_id
				LEFT JOIN st_users stu2 ON stu2.id=atb.owner_id
				WHERE atb.owner_id = $1
				GROUP BY att.id, att.trip_name, atb.id, stu2.name, ebs.status
				ORDER BY att.trip_name, atb.id`
