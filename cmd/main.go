package main

import (
	"fmt"

	"napredni_algoritmi_projekat/internal/config"
)

func main() {
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		fmt.Println("Greska:", err)
		return
	}

	fmt.Println("Konfiguracija uspesno ucitana:")
	fmt.Println("Memtable size:", cfg.MemtableSize)
	fmt.Println("WAL segment size:", cfg.WALSegmentSize)
	fmt.Println("Block size:", cfg.BlockSize)
	fmt.Println("Block cache size:", cfg.BlockCacheSize)
}
