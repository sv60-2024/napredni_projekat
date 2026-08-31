package sstable

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type IndexEntry struct {
	Key    string
	Offset uint64
}

func serializeIndexEntry(entry IndexEntry) ([]byte, error) {
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

func deserializeIndexEntry(data []byte) (IndexEntry, error) {
	buffer := bytes.NewReader(data)

	var keySize uint32
	if err := binary.Read(buffer, binary.LittleEndian, &keySize); err != nil {
		return IndexEntry{}, err
	}

	var offset uint64
	if err := binary.Read(buffer, binary.LittleEndian, &offset); err != nil {
		return IndexEntry{}, err
	}

	if uint64(keySize) > uint64(buffer.Len()) {
		return IndexEntry{}, errors.New("neispravan index zapis")
	}

	keyBytes := make([]byte, keySize)

	if _, err := buffer.Read(keyBytes); err != nil {
		return IndexEntry{}, err
	}

	return IndexEntry{
		Key:    string(keyBytes),
		Offset: offset,
	}, nil
}
