package record

type Record struct {
	Key       string
	Value     []byte
	Timestamp uint64
	Tombstone bool
}
