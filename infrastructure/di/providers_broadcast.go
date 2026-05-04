package di

import "flag-gen/infrastructure/broadcast"

func ProvideBroadcastHub() *broadcast.Hub {
	return broadcast.NewHub()
}
