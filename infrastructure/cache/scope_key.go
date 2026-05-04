package cache

import (
	"fmt"

	"flag-gen/domain/valueobject"
)

func ScopeKey(scope valueobject.Scope) string {
	return fmt.Sprintf("project:%s:env:%s", scope.ProjectID, scope.EnvironmentKey)
}
