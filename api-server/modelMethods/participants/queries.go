package participants

const (
	qryGetAll = `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
					FROM at_booking_people bp
						JOIN st_users p ON p.id=bp.person_id JOIN at_bookings b ON b.id=bp.booking_id WHERE (b.owner_id=$1 OR LOWER($2)='any')`
	qryGet = `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
				FROM at_booking_people bp
					JOIN st_users p ON p.id=bp.person_id JOIN at_bookings b ON b.id=bp.booking_id
				WHERE bp.id = $1 AND (b.owner_id=$2 OR LOWER($3)='any')`
	qryGetList = `SELECT bp.id, bp.owner_id, bp.booking_id, bp.person_id, p.name as person_name, bp.notes, bp.created, bp.modified
				FROM at_booking_people bp
					JOIN st_users p ON p.id=bp.person_id JOIN at_bookings b ON b.id=bp.booking_id
				WHERE bp.booking_id = $1 AND (b.owner_id=$2 OR LOWER($3)='any')`
	qryCreate = `INSERT INTO at_booking_people (owner_id, booking_id, person_id, notes) SELECT $1,b.id,$3,$4 FROM at_bookings b WHERE b.id=$2 AND (b.owner_id=$1 OR LOWER($5)='any') AND b.booking_status_id=1 AND b.stripe_session_id IS NULL AND NOT EXISTS (SELECT 1 FROM at_payments p WHERE p.booking_id=b.id) FOR UPDATE OF b RETURNING id`
	qryUpdate = `UPDATE at_booking_people
					SET (person_id, notes) = ($4, $5)
					WHERE id = $1 AND EXISTS (SELECT 1 FROM at_bookings b WHERE b.id=at_booking_people.booking_id AND (b.owner_id=$2 OR LOWER($3)='any') AND b.booking_status_id=1 AND b.stripe_session_id IS NULL AND NOT EXISTS (SELECT 1 FROM at_payments p WHERE p.booking_id=b.id) FOR UPDATE OF b)`
	qryDelete = `DELETE FROM at_booking_people WHERE id = $1 AND EXISTS (SELECT 1 FROM at_bookings b WHERE b.id=at_booking_people.booking_id AND (b.owner_id=$2 OR LOWER($3)='any') AND b.booking_status_id=1 AND b.stripe_session_id IS NULL AND NOT EXISTS (SELECT 1 FROM at_payments p WHERE p.booking_id=b.id) FOR UPDATE OF b)`
)
const CreateQuery = qryCreate
