package blockmanager

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.data")

	bm, err := New(BlockSize4KB, 10)
	if err != nil {
		t.Fatalf("neocekivana greska: %v", err)
	}

	payload := []byte("zdravo svete")
	if err := bm.WriteBlock(path, 0, payload); err != nil {
		t.Fatalf("WriteBlock nije uspeo: %v", err)
	}

	got, err := bm.ReadBlock(path, 0)
	if err != nil {
		t.Fatalf("ReadBlock nije uspeo: %v", err)
	}

	if len(got) != BlockSize4KB {
		t.Fatalf("ocekivana duzina bloka %d, dobijeno %d", BlockSize4KB, len(got))
	}

	if !bytes.Equal(got[:len(payload)], payload) {
		t.Fatalf("sadrzaj bloka se ne poklapa: dobijeno %q", got[:len(payload)])
	}

	for _, b := range got[len(payload):] {
		if b != 0 {
			t.Fatalf("padding nije popunjen nulama")
		}
	}
}

func TestWriteBlockRejectsOversizedData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.data")

	bm, _ := New(BlockSize4KB, 10)

	tooBig := make([]byte, BlockSize4KB+1)
	if err := bm.WriteBlock(path, 0, tooBig); err == nil {
		t.Fatalf("ocekivana greska za podatke vece od velicine bloka")
	}
}

func TestReadBlockOutOfRange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.data")

	bm, _ := New(BlockSize4KB, 10)

	if err := bm.WriteBlock(path, 0, []byte("prvi")); err != nil {
		t.Fatalf("WriteBlock nije uspeo: %v", err)
	}

	if _, err := bm.ReadBlock(path, 5); err == nil {
		t.Fatalf("ocekivana greska za citanje nepostojeceg bloka")
	}
}

func TestMultipleBlocksInSameFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.data")

	bm, _ := New(BlockSize4KB, 10)

	if err := bm.WriteBlock(path, 0, []byte("prvi")); err != nil {
		t.Fatalf("WriteBlock(0) nije uspeo: %v", err)
	}
	if err := bm.WriteBlock(path, 1, []byte("drugi")); err != nil {
		t.Fatalf("WriteBlock(1) nije uspeo: %v", err)
	}

	first, err := bm.ReadBlock(path, 0)
	if err != nil {
		t.Fatalf("ReadBlock(0) nije uspeo: %v", err)
	}
	second, err := bm.ReadBlock(path, 1)
	if err != nil {
		t.Fatalf("ReadBlock(1) nije uspeo: %v", err)
	}

	if !bytes.HasPrefix(first, []byte("prvi")) {
		t.Fatalf("sadrzaj prvog bloka se ne poklapa")
	}
	if !bytes.HasPrefix(second, []byte("drugi")) {
		t.Fatalf("sadrzaj drugog bloka se ne poklapa")
	}
}

func TestReadUsesCacheWithoutHittingDiskAgain(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.data")

	bm, _ := New(BlockSize4KB, 10)
	if err := bm.WriteBlock(path, 0, []byte("keshirano")); err != nil {
		t.Fatalf("WriteBlock nije uspeo: %v", err)
	}

	// Blok jos nije u kesu (write ga ne ubacuje ako ranije nije bio prisutan - spec 1.6),
	// pa prvo citanje popunjava kes (cache miss -> citanje sa diska -> upis u kes).
	if _, err := bm.ReadBlock(path, 0); err != nil {
		t.Fatalf("prvo ReadBlock (popunjavanje kesa) nije uspelo: %v", err)
	}

	// Obrisemo fajl sa diska - sad je blok vec u kesu, pa citanje i dalje treba da uspe.
	if err := os.Remove(path); err != nil {
		t.Fatalf("brisanje fajla nije uspelo: %v", err)
	}

	got, err := bm.ReadBlock(path, 0)
	if err != nil {
		t.Fatalf("ocekivano citanje iz kesa, dobijena greska: %v", err)
	}

	if !bytes.HasPrefix(got, []byte("keshirano")) {
		t.Fatalf("sadrzaj iz kesa se ne poklapa")
	}
}

func TestWriteUpdatesStaleCacheEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.data")

	bm, _ := New(BlockSize4KB, 10)

	if err := bm.WriteBlock(path, 0, []byte("staro")); err != nil {
		t.Fatalf("prvi WriteBlock nije uspeo: %v", err)
	}
	if _, err := bm.ReadBlock(path, 0); err != nil { // ucitaj u kes
		t.Fatalf("ReadBlock nije uspeo: %v", err)
	}

	if err := bm.WriteBlock(path, 0, []byte("novo")); err != nil {
		t.Fatalf("drugi WriteBlock nije uspeo: %v", err)
	}

	got, err := bm.ReadBlock(path, 0)
	if err != nil {
		t.Fatalf("ReadBlock nije uspeo: %v", err)
	}

	if !bytes.HasPrefix(got, []byte("novo")) {
		t.Fatalf("kes sadrzi zastarelu vrednost: %q", got[:4])
	}
}

func TestWriteDoesNotForceInsertIntoCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.data")

	bm, _ := New(BlockSize4KB, 10)

	// Blok nikad nije bio citan, pa ne sme biti u kesu ni posle pisanja
	// (spec 1.6: kes se azurira "ako je blok tu bio prisutan").
	if err := bm.WriteBlock(path, 0, []byte("prvi upis")); err != nil {
		t.Fatalf("WriteBlock nije uspeo: %v", err)
	}

	if bm.cache.Len() != 0 {
		t.Fatalf("write ne bi trebalo da ubaci blok u kes ako tamo ranije nije bio prisutan, a kes ima %d stavki", bm.cache.Len())
	}
}

func TestInvalidBlockSize(t *testing.T) {
	if _, err := New(1234, 10); err == nil {
		t.Fatalf("ocekivana greska za nevalidnu velicinu bloka")
	}
}

func TestInvalidCacheCapacity(t *testing.T) {
	if _, err := New(BlockSize4KB, 0); err == nil {
		t.Fatalf("ocekivana greska za nevalidnu velicinu kesa")
	}
}

func TestBlockCacheLRUEviction(t *testing.T) {
	cache := NewBlockCache(2)

	cache.Put("f", 0, []byte("a"))
	cache.Put("f", 1, []byte("b"))
	cache.Put("f", 2, []byte("c")) // treba da izbaci blok 0 (najdavnije koriscen)

	if _, ok := cache.Get("f", 0); ok {
		t.Fatalf("blok 0 je trebalo da bude izbacen iz kesa")
	}
	if _, ok := cache.Get("f", 1); !ok {
		t.Fatalf("blok 1 je trebalo da ostane u kesu")
	}
	if _, ok := cache.Get("f", 2); !ok {
		t.Fatalf("blok 2 je trebalo da ostane u kesu")
	}
}

func TestBlockCacheGetRefreshesRecency(t *testing.T) {
	cache := NewBlockCache(2)

	cache.Put("f", 0, []byte("a"))
	cache.Put("f", 1, []byte("b"))

	cache.Get("f", 0) // blok 0 postaje najskorije koriscen

	cache.Put("f", 2, []byte("c")) // sada treba da izbaci blok 1, ne blok 0

	if _, ok := cache.Get("f", 0); !ok {
		t.Fatalf("blok 0 nije trebalo da bude izbacen (skoro koriscen)")
	}
	if _, ok := cache.Get("f", 1); ok {
		t.Fatalf("blok 1 je trebalo da bude izbacen iz kesa")
	}
}

func TestBlockCacheGetReturnsIndependentCopy(t *testing.T) {
	cache := NewBlockCache(2)
	cache.Put("f", 0, []byte("abc"))

	got, _ := cache.Get("f", 0)
	got[0] = 'X'

	again, _ := cache.Get("f", 0)
	if again[0] != 'a' {
		t.Fatalf("izmena vracenog slice-a je uticala na interni sadrzaj kesa")
	}
}

func TestDifferentFilesDoNotShareBlockNumbers(t *testing.T) {
	dir := t.TempDir()
	pathA := filepath.Join(dir, "a.data")
	pathB := filepath.Join(dir, "b.data")

	bm, _ := New(BlockSize4KB, 10)

	if err := bm.WriteBlock(pathA, 0, []byte("A")); err != nil {
		t.Fatalf("WriteBlock A nije uspeo: %v", err)
	}
	if err := bm.WriteBlock(pathB, 0, []byte("B")); err != nil {
		t.Fatalf("WriteBlock B nije uspeo: %v", err)
	}

	gotA, _ := bm.ReadBlock(pathA, 0)
	gotB, _ := bm.ReadBlock(pathB, 0)

	if !bytes.HasPrefix(gotA, []byte("A")) || !bytes.HasPrefix(gotB, []byte("B")) {
		t.Fatalf("blokovi razlicitih fajlova su pomeseni u kesu")
	}
}
