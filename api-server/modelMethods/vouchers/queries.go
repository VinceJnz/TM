package vouchers

const (
	qryGetAllActive = `SELECT id, code, discount_percent, fixed_cost, expiry_date, is_active, created, modified
					FROM at_booking_vouchers
					WHERE is_active = true
					AND expiry_date > (CURRENT_TIMESTAMP AT TIME ZONE 'Pacific/Auckland')::date
					ORDER BY code`

	qryGetAllAdmin = `SELECT id, code, discount_percent, fixed_cost, expiry_date, is_active, created, modified
					FROM at_booking_vouchers
					ORDER BY code`

	qryGet = `SELECT id, code, discount_percent, fixed_cost, expiry_date, is_active, created, modified
				FROM at_booking_vouchers
				WHERE id = $1`

	qryCreate = `INSERT INTO at_booking_vouchers (code, discount_percent, fixed_cost, expiry_date, is_active)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id`

	qryUpdate = `UPDATE at_booking_vouchers
				SET code = $1, discount_percent = $2, fixed_cost = $3, expiry_date = $4, is_active = $5
				WHERE id = $6`

	qryDelete = `DELETE FROM at_booking_vouchers WHERE id = $1`
)
