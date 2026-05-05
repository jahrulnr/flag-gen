// nolint

package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
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

func (f *fakeL1) Delete(k string) {
	delete(f.m, k)
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

func (f *fakeL2) Del(_ context.Context, k string) error {
	delete(f.m, k)
	return nil
}

type fakeRepo struct {
	flags []entity.Flag
	err   error
}

func (f *fakeRepo) FindByKey(context.Context, valueobject.Scope, string) (*entity.Flag, error) {
	return nil, errors.New("unused")
}
func (f *fakeRepo) ListByScope(context.Context, valueobject.Scope, string, int, int) ([]entity.Flag, int, error) {
	if f.err != nil {
		return nil, 0, f.err
	}
	return f.flags, len(f.flags), nil
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

type fakeSnapMetaRepo struct {
	markInvalid int
}

var _ repository.SnapshotMetadataRepository = (*fakeSnapMetaRepo)(nil)

func (f *fakeSnapMetaRepo) GetByScope(context.Context, valueobject.Scope) (*entity.SnapshotMetadata, error) {
	return nil, errors.New("unused")
}
func (f *fakeSnapMetaRepo) Upsert(context.Context, entity.SnapshotMetadata) error {
	return nil
}
func (f *fakeSnapMetaRepo) MarkInvalidByScope(context.Context, valueobject.Scope) error {
	f.markInvalid++
	return nil
}

func (f *fakeSnapMetaRepo) ListScopes(context.Context) ([]valueobject.Scope, error) {
	return nil, nil
}

func TestScopedFlagCache_L1Hit(t *testing.T) {
	l1 := &fakeL1{m: map[string][]entity.Flag{"project:p1:env:prod": {{Key: "k1"}}}}
	l2 := &fakeL2{m: map[string][]entity.Flag{}}
	repo := &fakeRepo{flags: []entity.Flag{{Key: "k2"}}}

	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, nil, nil, nil, nil)
	got, err := c.Get(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"})
	require.NoError(t, err)
	require.Equal(t, "k1", got[0].Key)
}

func TestScopedFlagCache_L2HitRefillsL1(t *testing.T) {
	l1 := &fakeL1{m: map[string][]entity.Flag{}}
	l2 := &fakeL2{m: map[string][]entity.Flag{"project:p1:env:prod": {{Key: "k1"}}}}
	repo := &fakeRepo{flags: []entity.Flag{{Key: "k2"}}}

	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, nil, nil, nil, nil)
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

	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, nil, nil, nil, nil)
	got, err := c.Get(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"})
	require.NoError(t, err)
	require.Equal(t, "kdb", got[0].Key)
	require.NotEmpty(t, l1.m["project:p1:env:prod"])
	require.NotEmpty(t, l2.m["project:p1:env:prod"])
}

func TestScopedFlagCache_DBFailureUsesSnapshotAndShortTTL(t *testing.T) {
	dir := t.TempDir()
	w := snapshot.NewWriter(dir)
	_, _, err := w.WriteAtomic("p1", "prod", 42, []map[string]any{{"key": "kfile"}})
	require.NoError(t, err)

	l1 := &fakeL1{m: map[string][]entity.Flag{}}
	l2 := &fakeL2{m: map[string][]entity.Flag{}}
	repo := &fakeRepo{err: errors.New("db down")}

	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, snapshot.NewReader(dir), nil, nil, nil)
	got, err := c.Get(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"})
	require.NoError(t, err)
	require.Equal(t, "kfile", got[0].Key)
}

func TestScopedFlagCache_ChecksumMismatchMarksMetadataInvalid(t *testing.T) {
	dir := t.TempDir()
	w := snapshot.NewWriter(dir)
	path, _, err := w.WriteAtomic("p1", "prod", 3, []map[string]any{{"key": "kfile"}})
	require.NoError(t, err)

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var f snapshot.File
	require.NoError(t, json.Unmarshal(b, &f))
	f.Checksum.Hex = "badchecksum"
	out, err := json.MarshalIndent(f, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, out, 0o644))

	l1 := &fakeL1{m: map[string][]entity.Flag{}}
	l2 := &fakeL2{m: map[string][]entity.Flag{}}
	repo := &fakeRepo{err: errors.New("db down")}
	meta := &fakeSnapMetaRepo{}

	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, snapshot.NewReader(dir), meta, nil, nil)
	_, err = c.Get(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"})
	require.Error(t, err)
	require.Equal(t, 1, meta.markInvalid)
}

func TestScopedFlagCache_DegradedWhenNoDBNoSnapshot(t *testing.T) {
	l1 := &fakeL1{m: map[string][]entity.Flag{}}
	l2 := &fakeL2{m: map[string][]entity.Flag{}}
	repo := &fakeRepo{err: errors.New("db down")}

	// Point snapshot reader at empty dir (file missing)
	c := NewScopedFlagCache(l1, l2, NewSingleflight(), repo, snapshot.NewReader(os.TempDir()), nil, nil, nil)
	_, err := c.Get(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "does-not-exist"})
	require.Error(t, err)
}
