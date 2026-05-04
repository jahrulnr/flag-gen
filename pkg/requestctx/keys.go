package requestctx

// ctxKey is unexported to avoid collisions.
type ctxKey string

const (
	CorrelationIDKey ctxKey = "correlationId"
)

