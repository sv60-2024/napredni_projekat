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

	// Kasnije ovde ide:
	// WAL.Append(rec)

	e.memtable.Put(rec)

	// Kasnije ovde proveravamo flush u SSTable.
	if e.memtable.IsFull() {
		// SSTable.Create(e.memtable.GetAllSorted())
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

	// Kasnije ovde:
	// SSTable.Get(key)

	return nil, errors.New("kljuc ne postoji")
}

func (e *Engine) Delete(key string) error {
	if key == "" {
		return errors.New("kljuc ne sme biti prazan")
	}

	timestamp := uint64(time.Now().Unix())

	// Kasnije prvo ide WAL DELETE zapis.

	e.memtable.Delete(key, timestamp)

	return nil
}
