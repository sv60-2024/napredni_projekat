package sstable

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"napredni_algoritmi_projekat/internal/blockmanager"
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

func (s *SSTable) Save(directory string, id int, bm *blockmanager.BlockManager) error {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}

	summaryData, err := serializeSummary(s.Summary)
	if err != nil {
		return err
	}

	filterData := serializeBloomFilter(s.Filter)

	files := map[string][]byte{
		filepath.Join(directory, fmt.Sprintf("%06d-data.db", id)):    s.Data,
		filepath.Join(directory, fmt.Sprintf("%06d-index.db", id)):   s.Index,
		filepath.Join(directory, fmt.Sprintf("%06d-summary.db", id)): summaryData,
		filepath.Join(directory, fmt.Sprintf("%06d-filter.db", id)):  filterData,
	}

	for path, data := range files {
		if err := writeBytes(path, data, bm); err != nil {
			return err
		}
	}

	return nil
}

func LoadAll(directory string, bm *blockmanager.BlockManager) ([]*SSTable, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	ids := make([]int, 0)

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() || !strings.HasSuffix(name, "-data.db") {
			continue
		}

		idText := strings.TrimSuffix(name, "-data.db")

		id, err := strconv.Atoi(idText)
		if err != nil {
			continue
		}

		ids = append(ids, id)
	}

	sort.Ints(ids)

	tables := make([]*SSTable, 0, len(ids))

	for _, id := range ids {
		data, err := readBytes(
			filepath.Join(directory, fmt.Sprintf("%06d-data.db", id)),
			bm,
		)
		if err != nil {
			return nil, err
		}

		index, err := readBytes(
			filepath.Join(directory, fmt.Sprintf("%06d-index.db", id)),
			bm,
		)
		if err != nil {
			return nil, err
		}

		summaryData, err := readBytes(
			filepath.Join(directory, fmt.Sprintf("%06d-summary.db", id)),
			bm,
		)
		if err != nil {
			return nil, err
		}

		filterData, err := readBytes(
			filepath.Join(directory, fmt.Sprintf("%06d-filter.db", id)),
			bm,
		)
		if err != nil {
			return nil, err
		}

		summary, err := deserializeSummary(summaryData)
		if err != nil {
			return nil, err
		}

		filter, err := deserializeBloomFilter(filterData)
		if err != nil {
			return nil, err
		}

		tables = append(tables, &SSTable{
			Data:    data,
			Index:   index,
			Summary: summary,
			Filter:  filter,
		})
	}

	return tables, nil
}

func NextID(directory string) (int, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return 0, err
	}

	entries, err := os.ReadDir(directory)
	if err != nil {
		return 0, err
	}

	maxID := 0

	for _, entry := range entries {
		name := entry.Name()

		if entry.IsDir() || !strings.HasSuffix(name, "-data.db") {
			continue
		}

		idText := strings.TrimSuffix(name, "-data.db")

		id, err := strconv.Atoi(idText)
		if err == nil && id > maxID {
			maxID = id
		}
	}

	return maxID + 1, nil
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

func writeBytes(
	path string,
	data []byte,
	bm *blockmanager.BlockManager,
) error {
	payload := make([]byte, 8+len(data))

	binary.LittleEndian.PutUint64(
		payload[:8],
		uint64(len(data)),
	)

	copy(payload[8:], data)

	blockSize := bm.BlockSize()
	blockNumber := 0

	for start := 0; start < len(payload); start += blockSize {
		end := start + blockSize

		if end > len(payload) {
			end = len(payload)
		}

		if err := bm.WriteBlock(
			path,
			blockNumber,
			payload[start:end],
		); err != nil {
			return err
		}

		blockNumber++
	}

	return nil
}

func readBytes(
	path string,
	bm *blockmanager.BlockManager,
) ([]byte, error) {
	firstBlock, err := bm.ReadBlock(path, 0)
	if err != nil {
		return nil, err
	}

	if len(firstBlock) < 8 {
		return nil, errors.New("neispravan SSTable fajl")
	}

	dataSize := binary.LittleEndian.Uint64(firstBlock[:8])

	totalSize := uint64(8) + dataSize
	blockSize := uint64(bm.BlockSize())

	blockCount := int(
		(totalSize + blockSize - 1) / blockSize,
	)

	all := make(
		[]byte,
		0,
		blockCount*bm.BlockSize(),
	)

	all = append(all, firstBlock...)

	for blockNumber := 1; blockNumber < blockCount; blockNumber++ {
		block, err := bm.ReadBlock(path, blockNumber)
		if err != nil {
			return nil, err
		}

		all = append(all, block...)
	}

	end := 8 + int(dataSize)

	if end > len(all) {
		return nil, errors.New("neispravan SSTable fajl")
	}

	result := make([]byte, dataSize)
	copy(result, all[8:end])

	return result, nil
}

func serializeSummary(summary *Summary) ([]byte, error) {
	buffer := new(bytes.Buffer)

	if err := writeString(buffer, summary.MinKey); err != nil {
		return nil, err
	}

	if err := writeString(buffer, summary.MaxKey); err != nil {
		return nil, err
	}

	if err := binary.Write(
		buffer,
		binary.LittleEndian,
		uint32(len(summary.Entries)),
	); err != nil {
		return nil, err
	}

	for _, entry := range summary.Entries {
		data, err := serializeSummaryEntry(entry)
		if err != nil {
			return nil, err
		}

		if err := binary.Write(
			buffer,
			binary.LittleEndian,
			uint32(len(data)),
		); err != nil {
			return nil, err
		}

		if _, err := buffer.Write(data); err != nil {
			return nil, err
		}
	}

	return buffer.Bytes(), nil
}

func deserializeSummary(data []byte) (*Summary, error) {
	reader := bytes.NewReader(data)

	minKey, err := readString(reader)
	if err != nil {
		return nil, err
	}

	maxKey, err := readString(reader)
	if err != nil {
		return nil, err
	}

	var count uint32

	if err := binary.Read(
		reader,
		binary.LittleEndian,
		&count,
	); err != nil {
		return nil, err
	}

	entries := make([]SummaryEntry, 0, count)

	for i := uint32(0); i < count; i++ {
		var size uint32

		if err := binary.Read(
			reader,
			binary.LittleEndian,
			&size,
		); err != nil {
			return nil, err
		}

		entryData := make([]byte, size)

		if _, err := io.ReadFull(reader, entryData); err != nil {
			return nil, err
		}

		entry, err := deserializeSummaryEntry(entryData)
		if err != nil {
			return nil, err
		}

		entries = append(entries, entry)
	}

	return &Summary{
		MinKey:  minKey,
		MaxKey:  maxKey,
		Entries: entries,
	}, nil
}

func serializeBloomFilter(filter *BloomFilter) []byte {
	data := make([]byte, 8+len(filter.bits))

	binary.LittleEndian.PutUint64(
		data[:8],
		filter.size,
	)

	for i, bit := range filter.bits {
		if bit {
			data[8+i] = 1
		}
	}

	return data
}

func deserializeBloomFilter(data []byte) (*BloomFilter, error) {
	if len(data) < 8 {
		return nil, errors.New("neispravan bloom filter")
	}

	size := binary.LittleEndian.Uint64(data[:8])

	if size == 0 || uint64(len(data)-8) < size {
		return nil, errors.New("neispravan bloom filter")
	}

	filter := NewBloomFilter(size)

	for i := uint64(0); i < size; i++ {
		filter.bits[i] = data[8+int(i)] == 1
	}

	return filter, nil
}

func writeString(
	buffer *bytes.Buffer,
	value string,
) error {
	data := []byte(value)

	if err := binary.Write(
		buffer,
		binary.LittleEndian,
		uint32(len(data)),
	); err != nil {
		return err
	}

	_, err := buffer.Write(data)
	return err
}

func readString(reader *bytes.Reader) (string, error) {
	var size uint32

	if err := binary.Read(
		reader,
		binary.LittleEndian,
		&size,
	); err != nil {
		return "", err
	}

	data := make([]byte, size)

	if _, err := io.ReadFull(reader, data); err != nil {
		return "", err
	}

	return string(data), nil
}
