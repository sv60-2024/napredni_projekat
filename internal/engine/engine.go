package engine

import (
	"errors"
	"time"

	"napredni_algoritmi_projekat/internal/blockmanager"
	"napredni_algoritmi_projekat/internal/config"
	"napredni_algoritmi_projekat/internal/memtable"
	"napredni_algoritmi_projekat/internal/record"
	"napredni_algoritmi_projekat/internal/sstable"
	"napredni_algoritmi_projekat/internal/wal"
)

type Engine struct {
	memtable     *memtable.Memtable
	wal          *wal.WAL
	blockManager *blockmanager.BlockManager
	sstables     []*sstable.SSTable
}

func NewEngine(cfg config.Config) (*Engine, error) {
	mt := memtable.NewMemtable(cfg.MemtableSize)

	w, err := wal.New("data/wal", cfg.WALSegmentSize)
	if err != nil {
		return nil, err
	}

	bm, err := blockmanager.New(
		cfg.BlockSize,
		cfg.BlockCacheSize,
	)
	if err != nil {
		return nil, err
	}

	tables, err := sstable.LoadAll(
		"data/sstable",
		bm,
	)
	if err != nil {
		return nil, err
	}

	records, err := w.Recover()
	if err != nil {
		return nil, err
	}

	for _, rec := range records {
		mt.Put(rec)
	}

	return &Engine{
		memtable:     mt,
		wal:          w,
		blockManager: bm,
		sstables:     tables,
	}, nil
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

	if err := e.wal.Append(rec); err != nil {
		return err
	}

	e.memtable.Put(rec)

	if e.memtable.IsFull() {
		if err := e.flushMemtable(); err != nil {
			return err
		}
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

	for i := len(e.sstables) - 1; i >= 0; i-- {
		rec, found, err := e.sstables[i].Get(key)

		if err != nil {
			return nil, err
		}

		if found {
			if rec.Tombstone {
				return nil, errors.New("kljuc ne postoji")
			}

			return rec.Value, nil
		}
	}

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

	if err := e.wal.Append(rec); err != nil {
		return err
	}

	e.memtable.Put(rec)

	if e.memtable.IsFull() {
		if err := e.flushMemtable(); err != nil {
			return err
		}
	}

	return nil
}

func (e *Engine) flushMemtable() error {
	records := e.memtable.GetAllSorted()

	table, err := sstable.Create(records)
	if err != nil {
		return err
	}

	id, err := sstable.NextID("data/sstable")
	if err != nil {
		return err
	}

	if err := table.Save(
		"data/sstable",
		id,
		e.blockManager,
	); err != nil {
		return err
	}

	e.sstables = append(e.sstables, table)

	e.memtable.Clear()

	return nil
}
