package sstable

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type SummaryEntry struct {
	Key    string
	Offset uint64
}

type Summary struct {
	MinKey  string
	MaxKey  string
	Entries []SummaryEntry
}

func serializeSummaryEntry(entry SummaryEntry) ([]byte, error) {
	buffer := new(bytes.Buffer)

	keyBytes := []byte(entry.Key)
	keySize := uint32(len(keyBytes))

	if err := binary.Write(buffer, binary.LittleEndian, keySize); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.LittleEndian, entry.Offset); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(keyBytes); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func deserializeSummaryEntry(data []byte) (SummaryEntry, error) {
	buffer := bytes.NewReader(data)

	var keySize uint32
	if err := binary.Read(buffer, binary.LittleEndian, &keySize); err != nil {
		return SummaryEntry{}, err
	}

	var offset uint64
	if err := binary.Read(buffer, binary.LittleEndian, &offset); err != nil {
		return SummaryEntry{}, err
	}

	if uint64(keySize) > uint64(buffer.Len()) {
		return SummaryEntry{}, errors.New("neispravan summary zapis")
	}

	keyBytes := make([]byte, keySize)

	if _, err := buffer.Read(keyBytes); err != nil {
		return SummaryEntry{}, err
	}

	return SummaryEntry{
		Key:    string(keyBytes),
		Offset: offset,
	}, nil
}

func (s *Summary) ContainsKey(key string) bool {
	if s.MinKey == "" || s.MaxKey == "" {
		return false
	}

	return key >= s.MinKey && key <= s.MaxKey
}

func (s *Summary) FindStartOffset(key string) uint64 {
	var offset uint64 = 0

	for _, entry := range s.Entries {
		if entry.Key > key {
			break
		}

		offset = entry.Offset
	}

	return offset
}
