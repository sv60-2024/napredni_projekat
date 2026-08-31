package sstable

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"

	"napredni_algoritmi_projekat/internal/record"
)

type SSTable struct {
	Data    []byte
	Index   []byte
	Summary *Summary
	Filter  *BloomFilter
}

func Create(records []record.Record) (*SSTable, error) {
	if len(records) == 0 {
		return nil, errors.New("nije moguce napraviti SSTable bez zapisa")
	}

	sortedRecords := make([]record.Record, len(records))
	copy(sortedRecords, records)

	sort.Slice(sortedRecords, func(i, j int) bool {
		return sortedRecords[i].Key < sortedRecords[j].Key
	})

	dataBuffer := new(bytes.Buffer)
	indexBuffer := new(bytes.Buffer)

	summary := &Summary{
		MinKey:  sortedRecords[0].Key,
		MaxKey:  sortedRecords[len(sortedRecords)-1].Key,
		Entries: make([]SummaryEntry, 0),
	}

	filterSize := uint64(len(sortedRecords) * 10)
	if filterSize < 64 {
		filterSize = 64
	}

	filter := NewBloomFilter(filterSize)

	var dataOffset uint64
	var indexOffset uint64

	for i, rec := range sortedRecords {
		filter.Add(rec.Key)

		serializedRecord, err := serializeRecord(rec)
		if err != nil {
			return nil, err
		}

		if err := binary.Write(
			dataBuffer,
			binary.LittleEndian,
			uint32(len(serializedRecord)),
		); err != nil {
			return nil, err
		}

		if _, err := dataBuffer.Write(serializedRecord); err != nil {
			return nil, err
		}

		indexEntry := IndexEntry{
			Key:    rec.Key,
			Offset: dataOffset,
		}

		serializedIndex, err := serializeIndexEntry(indexEntry)
		if err != nil {
			return nil, err
		}

		if i%2 == 0 {
			summary.Entries = append(summary.Entries, SummaryEntry{
				Key:    rec.Key,
				Offset: indexOffset,
			})
		}

		if err := binary.Write(
			indexBuffer,
			binary.LittleEndian,
			uint32(len(serializedIndex)),
		); err != nil {
			return nil, err
		}

		if _, err := indexBuffer.Write(serializedIndex); err != nil {
			return nil, err
		}

		dataOffset += uint64(4 + len(serializedRecord))
		indexOffset += uint64(4 + len(serializedIndex))
	}

	return &SSTable{
		Data:    dataBuffer.Bytes(),
		Index:   indexBuffer.Bytes(),
		Summary: summary,
		Filter:  filter,
	}, nil
}
