package memtable

import (
	"sort"

	"napredni_algoritmi_projekat/internal/record"
)

type Memtable struct {
	records map[string]record.Record
	maxSize int
}

func NewMemtable(maxSize int) *Memtable {
	return &Memtable{
		records: make(map[string]record.Record),
		maxSize: maxSize,
	}
}

func (m *Memtable) Put(rec record.Record) {
	m.records[rec.Key] = rec
}

func (m *Memtable) Get(key string) (record.Record, bool) {
	rec, exists := m.records[key]
	return rec, exists
}

func (m *Memtable) IsFull() bool {
	return len(m.records) >= m.maxSize
}

func (m *Memtable) Size() int {
	return len(m.records)
}

func (m *Memtable) Clear() {
	m.records = make(map[string]record.Record)
}

func (m *Memtable) GetAllSorted() []record.Record {
	records := make([]record.Record, 0, len(m.records))

	for _, rec := range m.records {
		records = append(records, rec)
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].Key < records[j].Key
	})

	return records
}
