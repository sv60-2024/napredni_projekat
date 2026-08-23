package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"

	"napredni_algoritmi_projekat/internal/record"
)

const recordHeaderSize = 21

var errPartialRecord = fmt.Errorf("nepotpun WAL zapis")

func serializeRecord(rec record.Record) ([]byte, error) {
	keyBytes := []byte(rec.Key)
	valueBytes := rec.Value

	if len(keyBytes) == 0 {
		return nil, fmt.Errorf("kljuc ne sme biti prazan")
	}

	if len(keyBytes) > int(^uint32(0)) || len(valueBytes) > int(^uint32(0)) {
		return nil, fmt.Errorf("kljuc ili vrednost su preveliki za WAL zapis")
	}

	data := make([]byte, recordHeaderSize+len(keyBytes)+len(valueBytes))

	binary.LittleEndian.PutUint64(data[4:12], rec.Timestamp)
	if rec.Tombstone {
		data[12] = 1
	}
	binary.LittleEndian.PutUint32(data[13:17], uint32(len(keyBytes)))
	binary.LittleEndian.PutUint32(data[17:21], uint32(len(valueBytes)))

	copy(data[recordHeaderSize:], keyBytes)
	copy(data[recordHeaderSize+len(keyBytes):], valueBytes)

	crc := crc32.ChecksumIEEE(data[4:])
	binary.LittleEndian.PutUint32(data[0:4], crc)

	return data, nil
}

func deserializeRecord(reader io.Reader) (record.Record, int, error) {
	header := make([]byte, recordHeaderSize)
	read, err := io.ReadFull(reader, header)
	if err != nil {
		if err == io.EOF && read == 0 {
			return record.Record{}, 0, io.EOF
		}
		return record.Record{}, read, errPartialRecord
	}

	keySize := binary.LittleEndian.Uint32(header[13:17])
	valueSize := binary.LittleEndian.Uint32(header[17:21])
	bodySize := int(keySize) + int(valueSize)
	body := make([]byte, bodySize)

	read, err = io.ReadFull(reader, body)
	totalRead := recordHeaderSize + read
	if err != nil {
		return record.Record{}, totalRead, errPartialRecord
	}

	expectedCRC := binary.LittleEndian.Uint32(header[0:4])
	crcData := make([]byte, 0, recordHeaderSize-4+bodySize)
	crcData = append(crcData, header[4:]...)
	crcData = append(crcData, body...)

	actualCRC := crc32.ChecksumIEEE(crcData)
	if actualCRC != expectedCRC {
		return record.Record{}, totalRead, fmt.Errorf("CRC provera nije uspela")
	}

	keyEnd := int(keySize)
	rec := record.Record{
		Key:       string(body[:keyEnd]),
		Value:     append([]byte(nil), body[keyEnd:]...),
		Timestamp: binary.LittleEndian.Uint64(header[4:12]),
		Tombstone: header[12] == 1,
	}

	return rec, totalRead, nil
}
