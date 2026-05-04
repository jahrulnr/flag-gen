package migrate

import (
	"context"
	"database/sql"
	"fmt"
)

// TryAdvisoryLock acquires a named MySQL advisory lock (GET_LOCK).
// timeoutSec is the wait timeout passed to GET_LOCK.
func TryAdvisoryLock(ctx context.Context, db *sql.DB, lockName string, timeoutSec int) (bool, error) {
	var res sql.NullInt64
	err := db.QueryRowContext(ctx, `SELECT GET_LOCK(?, ?)`, lockName, timeoutSec).Scan(&res)
	if err != nil {
		return false, err
	}
	if !res.Valid {
		return false, fmt.Errorf("GET_LOCK returned NULL")
	}
	// 1 = success, 0 = timeout, NULL = error
	return res.Int64 == 1, nil
}

// ReleaseAdvisoryLock releases a MySQL advisory lock (RELEASE_LOCK).
func ReleaseAdvisoryLock(ctx context.Context, db *sql.DB, lockName string) {
	_, _ = db.ExecContext(ctx, `SELECT RELEASE_LOCK(?)`, lockName)
}
