package engine

import (
	"errors"
	"time"

	"napredni_algoritmi_projekat/internal/memtable"
	"napredni_algoritmi_projekat/internal/record"
)

type Engine struct {
	memtable *memtable.Memtable
}

func NewEngine(memtableSize int) *Engine {
	return &Engine{
		memtable: memtable.NewMemtable(memtableSize),
	}
}

func (e *Engine) Put(key string, value []byte) error {
	if key == "" {
		return errors.New("kljuc ne sme biti prazan")
	}

	rec := record.Record{
		Key:       key,
		Value:     value,
		Timestamp: uint64(time.Now().Unix()),
		Tombstone: false,
	}

	// Kasnije ce ovde prvo ici WAL zapis.
	// wal.Append(rec)

	e.memtable.Put(rec)

	if e.memtable.IsFull() {
		// Kasnije:
		// records := e.memtable.GetAllSorted()
		// sstable.Create(records)
		// e.memtable.Clear()
	}

	return nil
}

func (e *Engine) Get(key string) ([]byte, error) {
	if key == "" {
		return nil, errors.New("kljuc ne sme biti prazan")
	}

	rec, exists := e.memtable.Get(key)

	if exists {
		if rec.Tombstone {
			return nil, errors.New("kljuc ne postoji")
		}

		return rec.Value, nil
	}

	// Kasnije ovde ide pretraga SSTable-a.

	return nil, errors.New("kljuc ne postoji")
}

func (e *Engine) Delete(key string) error {
	if key == "" {
		return errors.New("kljuc ne sme biti prazan")
	}

	rec := record.Record{
		Key:       key,
		Value:     nil,
		Timestamp: uint64(time.Now().Unix()),
		Tombstone: true,
	}

	// Kasnije prvo ide WAL zapis.
	// wal.Append(rec)

	e.memtable.Put(rec)

	return nil
}
