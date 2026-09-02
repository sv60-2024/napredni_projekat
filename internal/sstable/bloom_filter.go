package sstable

import (
	"hash/fnv"
)

type BloomFilter struct {
	bits []bool
	size uint64
}

func NewBloomFilter(size uint64) *BloomFilter {
	return &BloomFilter{
		bits: make([]bool, size),
		size: size,
	}
}

func (bf *BloomFilter) Add(key string) {
	index1 := bf.hash1(key)
	index2 := bf.hash2(key)

	bf.bits[index1] = true
	bf.bits[index2] = true
}

func (bf *BloomFilter) MightContain(key string) bool {
	index1 := bf.hash1(key)
	index2 := bf.hash2(key)

	return bf.bits[index1] && bf.bits[index2]
}

func (bf *BloomFilter) hash1(key string) uint64 {
	h := fnv.New64a()
	h.Write([]byte(key))

	return h.Sum64() % bf.size
}

func (bf *BloomFilter) hash2(key string) uint64 {
	h := fnv.New64()
	h.Write([]byte(key))

	return h.Sum64() % bf.size
}
