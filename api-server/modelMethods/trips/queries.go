package trips

const (
	qryGetAll = `SELECT att.version,att.id,att.owner_id,att.trip_name,att.location,att.from_date,att.to_date,att.max_participants,att.created,att.modified,att.trip_status_id,att.trip_type_id,att.difficulty_level_id,att.trip_cost_group_id,att.description, ettd.level as difficulty_level, etts.status as trip_status, sum(atbcount.participants) as participants
					FROM at_trips att
					LEFT JOIN et_trip_difficulty ettd ON ettd.id=att.difficulty_level_id
					JOIN et_trip_status etts ON etts.id=att.trip_status_id
					LEFT JOIN (SELECT atb.id,atb.trip_id, COUNT(atb.id) as participants
						FROM at_bookings atb
						JOIN at_booking_people atbp ON atbp.booking_id=atb.id
						GROUP BY atb.id) atbcount ON atbcount.trip_id=att.id
					--WHERE att.owner_id = $1 OR true=$2
					GROUP BY att.id, ettd.level, etts.status`
	qryGet = `SELECT att.version,att.id,att.owner_id,att.trip_name,att.location,att.from_date,att.to_date,att.max_participants,att.created,att.modified,att.trip_status_id,att.trip_type_id,att.difficulty_level_id,att.trip_cost_group_id,att.description, etts.status as trip_status
					FROM at_trips att
					LEFT JOIN et_trip_status etts ON etts.id=att.trip_status_id
					WHERE att.id = $1`
	qryCreate = `INSERT INTO at_trips (owner_id, trip_name, location, difficulty_level_id, from_date, to_date, max_participants, trip_status_id, trip_type_id, trip_cost_group_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) RETURNING id`
	qryUpdate = `UPDATE at_trips SET (trip_name, location, difficulty_level_id, from_date, to_date, max_participants, trip_status_id, trip_type_id, trip_cost_group_id) = ($4, $5, $6, $7, $8, $9, $10, $11, $12) WHERE id = $1 AND (owner_id = $2 OR LOWER($3)='any')`
	qryDelete = `DELETE FROM at_trips WHERE id = $1 AND (owner_id = $2 OR LOWER($3)='any')`
)
