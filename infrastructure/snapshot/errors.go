package snapshot

import "errors"

// ErrChecksumMismatch is returned when snapshot JSON parses but checksum does not match canonical payload.
var ErrChecksumMismatch = errors.New("snapshot_checksum_mismatch")
