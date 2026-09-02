package sstable

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"

	"napredni_algoritmi_projekat/internal/record"
)

func serializeRecord(rec record.Record) ([]byte, error) {
	buffer := new(bytes.Buffer)

	keyBytes := []byte(rec.Key)

	if err := binary.Write(buffer, binary.LittleEndian, rec.Timestamp); err != nil {
		return nil, err
	}

	if err := binary.Write(buffer, binary.LittleEndian, rec.Tombstone); err != nil {
		return nil, err
	}

	keySize := uint32(len(keyBytes))
	if err := binary.Write(buffer, binary.LittleEndian, keySize); err != nil {
		return nil, err
	}

	valueSize := uint32(len(rec.Value))
	if err := binary.Write(buffer, binary.LittleEndian, valueSize); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(keyBytes); err != nil {
		return nil, err
	}

	if _, err := buffer.Write(rec.Value); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func deserializeRecord(data []byte) (record.Record, error) {
	buffer := bytes.NewReader(data)

	var timestamp uint64
	if err := binary.Read(buffer, binary.LittleEndian, &timestamp); err != nil {
		return record.Record{}, err
	}

	var tombstone bool
	if err := binary.Read(buffer, binary.LittleEndian, &tombstone); err != nil {
		return record.Record{}, err
	}

	var keySize uint32
	if err := binary.Read(buffer, binary.LittleEndian, &keySize); err != nil {
		return record.Record{}, err
	}

	var valueSize uint32
	if err := binary.Read(buffer, binary.LittleEndian, &valueSize); err != nil {
		return record.Record{}, err
	}

	if uint64(keySize)+uint64(valueSize) > uint64(buffer.Len()) {
		return record.Record{}, errors.New("neispravan SSTable zapis")
	}

	keyBytes := make([]byte, keySize)

	if _, err := io.ReadFull(buffer, keyBytes); err != nil {
		return record.Record{}, err
	}

	valueBytes := make([]byte, valueSize)

	if valueSize > 0 {
		if _, err := io.ReadFull(buffer, valueBytes); err != nil {
			return record.Record{}, err
		}
	}

	return record.Record{
		Key:       string(keyBytes),
		Value:     valueBytes,
		Timestamp: timestamp,
		Tombstone: tombstone,
	}, nil
}
