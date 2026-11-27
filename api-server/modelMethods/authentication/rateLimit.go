package authentication

import "context"

func (s Store) RecordRateLimitAttempt(ctx context.Context, keyHash string, seconds int) (int, error) {
	var attempts int
	err := s.DB.GetContext(ctx, &attempts, `WITH expired AS (DELETE FROM st_auth_rate_limits WHERE window_start<CURRENT_TIMESTAMP-INTERVAL '2 hours') INSERT INTO st_auth_rate_limits(key_hash,window_start,attempts) VALUES($1,to_timestamp(floor(extract(epoch FROM CURRENT_TIMESTAMP)/$2::numeric)*$2::numeric),1) ON CONFLICT(key_hash,window_start) DO UPDATE SET attempts=st_auth_rate_limits.attempts+1 RETURNING attempts`, keyHash, seconds)
	return attempts, err
}
