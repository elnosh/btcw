package utils

import "encoding/binary"

func Int64ToBytes(sats int64) []byte {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(sats))
	return b
}

func BytesToInt64(b []byte) int64 {
	return int64(binary.LittleEndian.Uint64(b))
}
