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

func (s *SSTable) Get(key string) (record.Record, bool, error) {
	if !s.Filter.MightContain(key) {
		return record.Record{}, false, nil
	}

	if !s.Summary.ContainsKey(key) {
		return record.Record{}, false, nil
	}

	indexOffset := s.Summary.FindStartOffset(key)

	for indexOffset < uint64(len(s.Index)) {
		entry, nextOffset, err := s.readIndexEntry(indexOffset)
		if err != nil {
			return record.Record{}, false, err
		}

		if entry.Key == key {
			rec, err := s.readDataRecord(entry.Offset)
			if err != nil {
				return record.Record{}, false, err
			}

			return rec, true, nil
		}

		if entry.Key > key {
			return record.Record{}, false, nil
		}

		indexOffset = nextOffset
	}

	return record.Record{}, false, nil
}

func (s *SSTable) readIndexEntry(offset uint64) (IndexEntry, uint64, error) {
	if offset+4 > uint64(len(s.Index)) {
		return IndexEntry{}, 0, errors.New("neispravan index offset")
	}

	size := binary.LittleEndian.Uint32(s.Index[offset : offset+4])

	start := offset + 4
	end := start + uint64(size)

	if end > uint64(len(s.Index)) {
		return IndexEntry{}, 0, errors.New("neispravan index zapis")
	}

	entry, err := deserializeIndexEntry(s.Index[start:end])
	if err != nil {
		return IndexEntry{}, 0, err
	}

	return entry, end, nil
}

func (s *SSTable) readDataRecord(offset uint64) (record.Record, error) {
	if offset+4 > uint64(len(s.Data)) {
		return record.Record{}, errors.New("neispravan data offset")
	}

	size := binary.LittleEndian.Uint32(s.Data[offset : offset+4])

	start := offset + 4
	end := start + uint64(size)

	if end > uint64(len(s.Data)) {
		return record.Record{}, errors.New("neispravan data zapis")
	}

	return deserializeRecord(s.Data[start:end])
}
