package wal

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"napredni_algoritmi_projekat/internal/record"
)

func TestAppendRecoverAndSegmentRotation(t *testing.T) {
	dir := t.TempDir()

	log, err := New(dir, 2)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	records := []record.Record{
		{Key: "a", Value: []byte("one"), Timestamp: 1},
		{Key: "b", Value: []byte("two"), Timestamp: 2},
		{Key: "c", Value: []byte("three"), Timestamp: 3, Tombstone: true},
	}

	for _, rec := range records {
		if err := log.Append(rec); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	segments, err := listSegments(dir)
	if err != nil {
		t.Fatalf("listSegments() error = %v", err)
	}

	if len(segments) != 2 {
		t.Fatalf("len(segments) = %d, want 2", len(segments))
	}

	recovered, err := log.Recover()
	if err != nil {
		t.Fatalf("Recover() error = %v", err)
	}

	if !reflect.DeepEqual(recovered, records) {
		t.Fatalf("Recover() = %+v, want %+v", recovered, records)
	}
}

func TestRecoverDetectsCRCError(t *testing.T) {
	dir := t.TempDir()

	log, err := New(dir, 10)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if err := log.Append(record.Record{Key: "a", Value: []byte("one"), Timestamp: 1}); err != nil {
		t.Fatalf("Append() error = %v", err)
	}

	path := segmentPath(dir, 0)
	file, err := os.OpenFile(path, os.O_RDWR, 0644)
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer file.Close()

	if _, err := file.Seek(0, 0); err != nil {
		t.Fatalf("Seek() error = %v", err)
	}

	if _, err := file.Write([]byte{0}); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	if _, err := log.Recover(); err == nil {
		t.Fatal("Recover() error = nil, want CRC error")
	}
}

func TestAppendRejectsRecordLargerThanBlock(t *testing.T) {
	dir := t.TempDir()

	log, err := New(dir, 10)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	value := make([]byte, log.blockManager.BlockSize())

	err = log.Append(record.Record{
		Key:       "oversized",
		Value:     value,
		Timestamp: 1,
	})
	if err == nil {
		t.Fatal("Append() error = nil, want oversized record error")
	}
}

func TestRemoveSegmentsUpTo(t *testing.T) {
	dir := t.TempDir()

	log, err := New(dir, 1)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for _, rec := range []record.Record{
		{Key: "a", Value: []byte("one"), Timestamp: 1},
		{Key: "b", Value: []byte("two"), Timestamp: 2},
		{Key: "c", Value: []byte("three"), Timestamp: 3},
	} {
		if err := log.Append(rec); err != nil {
			t.Fatalf("Append() error = %v", err)
		}
	}

	if err := log.RemoveSegmentsUpTo(1); err != nil {
		t.Fatalf("RemoveSegmentsUpTo() error = %v", err)
	}

	for _, name := range []string{"000000.wal", "000001.wal"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("%s still exists or stat returned unexpected error: %v", name, err)
		}
	}

	if _, err := os.Stat(filepath.Join(dir, "000002.wal")); err != nil {
		t.Fatalf("000002.wal missing: %v", err)
	}
}
