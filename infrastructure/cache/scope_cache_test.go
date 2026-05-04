// nolint

package cache

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/valueobject"
	"flag-gen/infrastructure/snapshot"

	"github.com/stretchr/testify/require"
)

type fakeL1 struct {
	m map[string][]entity.Flag
}

func (f *fakeL1) Get(k string) ([]entity.Flag, bool) {
	v, ok := f.m[k]
	return v, ok
}
func (f *fakeL1) Set(k string, v []entity.Flag, _ time.Duration) {
	f.m[k] = v
}

type fakeL2 struct {
	m   map[string][]entity.Flag
	err error
}

func (f *fakeL2) Get(_ context.Context, k string) ([]entity.Flag, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	v, ok := f.m[k]
	return v, ok, nil
}
func (f *fakeL2) Set(_ context.Context, k string, v []entity.Flag, _ time.Duration) error {
	if f.err != nil {
		return f.err
	}
	f.m[k] = v
	return nil
}

type fakeRepo struct {
	flags []entity.Flag
	err   error
}

func (f *fakeRepo) FindByKey(context.Context, valueobject.Scope, string) (*entity.Flag, error) {
	return nil, errors.New("unused")
}
func (f *fakeRepo) ListByScope(context.Context, valueobject.Scope) ([]entity.Flag, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.flags, nil
}

func (f *fakeRepo) FindByID(context.Context, valueobject.Scope, string) (*entity.Flag, error) {
	return nil, errors.New("unused")
}
func (f *fakeRepo) Create(context.Context, valueobject.Scope, entity.Flag) (entity.Flag, error) {
	return entity.Flag{}, errors.New("unused")
}
func (f *fakeRepo) Update(context.Context, valueobject.Scope, entity.Flag) (entity.Flag, error) {
	return entity.Flag{}, errors.New("unused")
}
func (f *fakeRepo) SoftDelete(context.Context, valueobject.Scope, string, int) error {
	return errors.New("unused")
}

func TestScopedFlagCache_L1Hit(t *testing.T) {
	l1 := &fakeL1{m: map[string][]entity.Flag{"project:p1:env:prod": {{Key: "k1"}}}}
	l2 := &fakeL2{m: map[string][]entity.Flag{}}
	repo := &fakeRepo{flags: []entity.Flag{{Key: "k2"}}}

	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, nil)
	got, err := c.Get(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"})
	require.NoError(t, err)
	require.Equal(t, "k1", got[0].Key)
}

func TestScopedFlagCache_L2HitRefillsL1(t *testing.T) {
	l1 := &fakeL1{m: map[string][]entity.Flag{}}
	l2 := &fakeL2{m: map[string][]entity.Flag{"project:p1:env:prod": {{Key: "k1"}}}}
	repo := &fakeRepo{flags: []entity.Flag{{Key: "k2"}}}

	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, nil)
	got, err := c.Get(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"})
	require.NoError(t, err)
	require.Equal(t, "k1", got[0].Key)
	_, ok := l1.m["project:p1:env:prod"]
	require.True(t, ok)
}

func TestScopedFlagCache_DBSuccessFillsCaches(t *testing.T) {
	l1 := &fakeL1{m: map[string][]entity.Flag{}}
	l2 := &fakeL2{m: map[string][]entity.Flag{}}
	repo := &fakeRepo{flags: []entity.Flag{{Key: "kdb"}}}

	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, nil)
	got, err := c.Get(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"})
	require.NoError(t, err)
	require.Equal(t, "kdb", got[0].Key)
	require.NotEmpty(t, l1.m["project:p1:env:prod"])
	require.NotEmpty(t, l2.m["project:p1:env:prod"])
}

func TestScopedFlagCache_DBFailureUsesSnapshotAndShortTTL(t *testing.T) {
	dir := t.TempDir()
	w := snapshot.NewWriter(dir)
	_, err := w.WriteAtomic("p1", "prod", 42, []map[string]any{{"key": "kfile"}})
	require.NoError(t, err)

	l1 := &fakeL1{m: map[string][]entity.Flag{}}
	l2 := &fakeL2{m: map[string][]entity.Flag{}}
	repo := &fakeRepo{err: errors.New("db down")}

	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, snapshot.NewReader(dir))
	got, err := c.Get(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"})
	require.NoError(t, err)
	require.Equal(t, "kfile", got[0].Key)
}

func TestScopedFlagCache_DegradedWhenNoDBNoSnapshot(t *testing.T) {
	l1 := &fakeL1{m: map[string][]entity.Flag{}}
	l2 := &fakeL2{m: map[string][]entity.Flag{}}
	repo := &fakeRepo{err: errors.New("db down")}

	// Point snapshot reader at empty dir (file missing)
	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, snapshot.NewReader(os.TempDir()))
	_, err := c.Get(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "does-not-exist"})
	require.Error(t, err)
}
