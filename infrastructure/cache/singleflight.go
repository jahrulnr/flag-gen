package cache

import "golang.org/x/sync/singleflight"

type Singleflight struct {
	g singleflight.Group
}

func NewSingleflight() *Singleflight {
	return &Singleflight{}
}

func (s *Singleflight) Do(scopeKey string, fn func() (any, error)) (any, error) {
	v, err, _ := s.g.Do(scopeKey, fn)
	return v, err
}
