package main

import (
	"fmt"
	"time"

	"napredni_algoritmi_projekat/internal/config"
	"napredni_algoritmi_projekat/internal/memtable"
	"napredni_algoritmi_projekat/internal/record"
)

func main() {
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		fmt.Println("Greska:", err)
		return
	}

	mt := memtable.NewMemtable(cfg.MemtableSize)

	rec1 := record.Record{
		Key:       "marko",
		Value:     []byte("Marko"),
		Timestamp: uint64(time.Now().Unix()),
		Tombstone: false,
	}

	rec2 := record.Record{
		Key:       "andrej",
		Value:     []byte("Andrej"),
		Timestamp: uint64(time.Now().Unix()),
		Tombstone: false,
	}

	mt.Put(rec1)
	mt.Put(rec2)

	rec, exists := mt.Get("andrej")
	if exists {
		fmt.Println("Pronadjen:", string(rec.Value))
	}

	fmt.Println("Broj elemenata:", mt.Size())
	fmt.Println("Memtable puna:", mt.IsFull())

	fmt.Println("Sortirani zapisi:")
	for _, r := range mt.GetAllSorted() {
		fmt.Println(r.Key, string(r.Value))
	}

	mt.Delete("andrej", uint64(time.Now().Unix()))

	deletedRec, exists := mt.Get("andrej")
	if exists {
		fmt.Println("Tombstone za andrej:", deletedRec.Tombstone)
	}
}
