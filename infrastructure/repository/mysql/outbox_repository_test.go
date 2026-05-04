// nolint

package mysql

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOutboxRepository_MethodsAreCallable(t *testing.T) {
	// This test only ensures the adapter compiles and methods are callable.
	// It does not require a live MySQL instance.
	db, err := NewDB("root:pass@tcp(localhost:3306)/db?parseTime=true")
	require.NoError(t, err)

	repo := NewOutboxRepository(db)

	_, _ = repo.ListPending(context.Background(), 10)
	_, _ = repo.ClaimNext(context.Background(), "w1", 10, 5_000_000_000)
	_ = repo.MarkPublished(context.Background(), "evt1")
	_ = repo.MarkFailed(context.Background(), "evt1", "oops", nil)
}
