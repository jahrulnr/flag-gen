package handler

import "time"

func responseNow() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

