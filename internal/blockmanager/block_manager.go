// Package blockmanager implementuje Block Manager i Block Cache (spec 1.6).
//
// Disk je segmentiran na blokove fiksne velicine. Block Manager je jedini sloj
// kroz koji ostatak sistema (WAL, Memtable flush, SSTable...) sme da cita i pise
// sadrzaj fajlova na disku - direktan pristup fajlovima mimo ovog sloja nije
// dozvoljen specifikacijom projekta.
package blockmanager

import (
	"fmt"
	"os"
	"sync"
)

// Dozvoljene velicine bloka, u bajtovima (spec 1.6).
const (
	BlockSize4KB  = 4096
	BlockSize8KB  = 8192
	BlockSize16KB = 16384
)

// BlockManager cita i pise blokove fiksne velicine u fajlovima na disku,
// oslanjajuci se na Block Cache da izbegne nepotrebne I/O operacije.
type BlockManager struct {
	blockSize int
	cache     *BlockCache
	mu        sync.Mutex
}

// New pravi novi BlockManager. blockSize mora biti 4096, 8192 ili 16384 (spec 1.6),
// a cacheCapacity je broj blokova koje Block Cache moze da drzi u memoriji.
func New(blockSize int, cacheCapacity int) (*BlockManager, error) {
	if err := validateBlockSize(blockSize); err != nil {
		return nil, err
	}

	if cacheCapacity <= 0 {
		return nil, fmt.Errorf("velicina block kesa mora biti veca od 0")
	}

	return &BlockManager{
		blockSize: blockSize,
		cache:     NewBlockCache(cacheCapacity),
	}, nil
}

func validateBlockSize(blockSize int) error {
	switch blockSize {
	case BlockSize4KB, BlockSize8KB, BlockSize16KB:
		return nil
	default:
		return fmt.Errorf("velicina bloka mora biti 4096, 8192 ili 16384 (dobijeno %d)", blockSize)
	}
}

// BlockSize vraca velicinu bloka kojom ovaj BlockManager rukuje.
func (bm *BlockManager) BlockSize() int {
	return bm.blockSize
}

// ReadBlock cita sadrzaj bloka sa rednim brojem blockNumber iz fajla na putanji path.
//
// Prvo se proverava Block Cache. Ako blok nije prisutan u kesu, cita se sa diska
// i potom upisuje u kes pre nego sto se vrati pozivaocu (spec 1.6). Povratni slice
// je uvek duzine BlockSize i predstavlja kopiju internog sadrzaja - izmena rezultata
// od strane pozivaoca ne utice na kes.
func (bm *BlockManager) ReadBlock(path string, blockNumber int) ([]byte, error) {
	if blockNumber < 0 {
		return nil, fmt.Errorf("redni broj bloka ne sme biti negativan")
	}

	bm.mu.Lock()
	defer bm.mu.Unlock()

	if data, ok := bm.cache.Get(path, blockNumber); ok {
		return data, nil
	}

	data, err := bm.readBlockFromDisk(path, blockNumber)
	if err != nil {
		return nil, err
	}

	bm.cache.Put(path, blockNumber, data)

	return data, nil
}

// WriteBlock upisuje data kao blok sa rednim brojem blockNumber u fajl na putanji path.
//
// Ako je data kraci od BlockSize, ostatak bloka se popunjava nulama (padding).
// Fajl se kreira ako jos ne postoji. Sadrzaj se upisuje na disk; ako je taj blok vec
// bio prisutan u Block Cache-u, njegov sadrzaj se azurira kako kes ne bi sadrzao
// zastarelu verziju - blok se ne ubacuje nasilno u kes ako tamo ranije nije bio (spec 1.6).
func (bm *BlockManager) WriteBlock(path string, blockNumber int, data []byte) error {
	if blockNumber < 0 {
		return fmt.Errorf("redni broj bloka ne sme biti negativan")
	}

	if len(data) > bm.blockSize {
		return fmt.Errorf("sadrzaj bloka (%d bajtova) je veci od velicine bloka (%d bajtova)", len(data), bm.blockSize)
	}

	padded := make([]byte, bm.blockSize)
	copy(padded, data)

	bm.mu.Lock()
	defer bm.mu.Unlock()

	if err := bm.writeBlockToDisk(path, blockNumber, padded); err != nil {
		return err
	}

	bm.cache.UpdateIfPresent(path, blockNumber, padded)

	return nil
}

func (bm *BlockManager) readBlockFromDisk(path string, blockNumber int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("greska pri otvaranju fajla %s: %w", path, err)
	}
	defer file.Close()

	offset := int64(blockNumber) * int64(bm.blockSize)

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("greska pri citanju informacija o fajlu %s: %w", path, err)
	}

	if offset+int64(bm.blockSize) > info.Size() {
		return nil, fmt.Errorf("blok %d ne postoji u fajlu %s", blockNumber, path)
	}

	buf := make([]byte, bm.blockSize)
	if _, err := file.ReadAt(buf, offset); err != nil {
		return nil, fmt.Errorf("greska pri citanju bloka %d iz fajla %s: %w", blockNumber, path, err)
	}

	return buf, nil
}

func (bm *BlockManager) writeBlockToDisk(path string, blockNumber int, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("greska pri otvaranju fajla %s: %w", path, err)
	}
	defer file.Close()

	offset := int64(blockNumber) * int64(bm.blockSize)

	if _, err := file.WriteAt(data, offset); err != nil {
		return fmt.Errorf("greska pri upisu bloka %d u fajl %s: %w", blockNumber, path, err)
	}

	if err := file.Sync(); err != nil {
		return fmt.Errorf("greska pri sinhronizaciji fajla %s: %w", path, err)
	}

	return nil
}
