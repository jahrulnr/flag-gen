package cache

import "flag-gen/domain/valueobject"

func ScopeKey(scope valueobject.Scope) string {
	return scope.CacheKey()
}
