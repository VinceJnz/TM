package bookings

const (
	qryGetAll = `SELECT stu2.username AS owner_username, atb.id,atb.owner_id,atb.trip_id,atb.notes,atb.from_date,atb.to_date,atb.booking_status_id,atb.created,atb.modified,atb.booking_date,atb.group_booking_id,atb.booking_price,atb.payment_date,atb.amount_paid,atb.currency,atb.stripe_session_id, ebs.status, COUNT(stu.name) as participants,
			SUM(attc.amount) * (EXTRACT(EPOCH FROM (atb.to_date - atb.from_date)) / 86400) as booking_cost,
			--SUM(attc.amount) AS booking_cost,
			att.trip_name
		FROM at_trips att
			LEFT JOIN at_bookings atb ON atb.trip_id=att.id
			LEFT JOIN at_booking_people atbp ON atbp.booking_id=atb.id
				JOIN et_booking_status ebs on ebs.id=atb.booking_status_id
			LEFT JOIN st_users stu ON stu.id=atbp.person_id
			LEFT JOIN at_trip_cost_groups attcg ON attcg.id=att.trip_cost_group_id
			LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id=att.trip_cost_group_id
									AND attc.member_status_id=stu.member_status_id
									AND attc.user_age_group_id=stu.user_age_group_id
			JOIN st_users stu2 ON stu2.id=atb.owner_id
		WHERE atb.owner_id = $1 OR LOWER($2)= 'any'
		GROUP BY stu2.username, att.id, att.trip_name, atb.id, ebs.status
		ORDER BY stu2.username, att.trip_name, atb.id;`

	qryGet = `SELECT b.id,b.owner_id,b.trip_id,b.notes,b.from_date,b.to_date,b.booking_status_id,b.created,b.modified,b.booking_date,b.group_booking_id,b.booking_price,b.payment_date,b.amount_paid,b.currency,b.stripe_session_id, ebs.status FROM at_bookings b JOIN et_booking_status ebs ON ebs.id=b.booking_status_id WHERE b.id=$1 AND (b.owner_id=$2 OR LOWER($3)='any')`

	qryGetList = `SELECT stu2.username AS owner_username, atb.id,atb.owner_id,atb.trip_id,atb.notes,atb.from_date,atb.to_date,atb.booking_status_id,atb.created,atb.modified,atb.booking_date,atb.group_booking_id,atb.booking_price,atb.payment_date,atb.amount_paid,atb.currency,atb.stripe_session_id,
			ebs.status, COUNT(stu.name) as participants,
			--SUM(attc.amount) AS booking_cost,
			SUM(attc.amount) * (EXTRACT(EPOCH FROM (atb.to_date - atb.from_date)) / 86400) as booking_cost,
			att.trip_name
		FROM at_trips att
			LEFT JOIN at_bookings atb ON atb.trip_id=att.id
			LEFT JOIN at_booking_people atbp ON atbp.booking_id=atb.id
				JOIN et_booking_status ebs on ebs.id=atb.booking_status_id
			LEFT JOIN st_users stu ON stu.id=atbp.person_id
			LEFT JOIN at_trip_cost_groups attcg ON attcg.id=att.trip_cost_group_id
			LEFT JOIN at_trip_costs attc ON attc.trip_cost_group_id=att.trip_cost_group_id
									AND attc.member_status_id=stu.member_status_id
									AND attc.user_age_group_id=stu.user_age_group_id
			JOIN st_users stu2 ON stu2.id=atb.owner_id
		WHERE atb.trip_id = $1 AND (atb.owner_id=$2 OR LOWER($3)='any')
		GROUP BY stu2.username, att.id, att.trip_name, atb.id, ebs.status
		ORDER BY stu2.username, att.trip_name, atb.id`

	qryCreate      = `INSERT INTO at_bookings (owner_id,trip_id,notes,from_date,to_date,booking_status_id) VALUES ($1,$2,$3,$4,$5,1) RETURNING id`
	qryUpdateAdmin = `UPDATE at_bookings SET (owner_id,trip_id,notes,from_date,to_date,booking_price)=($2,$3,$4,$5,$6,$7) WHERE id=$1 AND booking_status_id=1 AND stripe_session_id IS NULL AND NOT EXISTS (SELECT 1 FROM at_payments p WHERE p.booking_id=at_bookings.id)`
	qryUpdate      = `UPDATE at_bookings SET (notes,from_date,to_date)=($4,$5,$6) WHERE id=$1 AND (owner_id=$2 OR LOWER($3)='any') AND booking_status_id=1 AND stripe_session_id IS NULL AND NOT EXISTS (SELECT 1 FROM at_payments p WHERE p.booking_id=at_bookings.id)`
	qryDelete      = `DELETE FROM at_bookings WHERE id=$1 AND (owner_id=$2 OR LOWER($3)='any') AND booking_status_id=1 AND stripe_session_id IS NULL AND NOT EXISTS (SELECT 1 FROM at_payments p WHERE p.booking_id=at_bookings.id)`
)
