package config

type Config struct {
	MemtableSize   int `json:"memtable_size"`
	WALSegmentSize int `json:"wal_segment_size"`
	BlockSize      int `json:"block_size"`
	BlockCacheSize int `json:"block_cache_size"`
}
