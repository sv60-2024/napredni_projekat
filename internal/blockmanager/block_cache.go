package blockmanager

import "container/list"

// blockKey jednoznacno identifikuje blok - putanja fajla i redni broj bloka unutar njega.
type blockKey struct {
	path        string
	blockNumber int
}

// cacheEntry je element koji se cuva u LRU redu.
type cacheEntry struct {
	key  blockKey
	data []byte
}

// BlockCache je LRU kes za blokove ucitane sa diska (spec 1.6).
// Nije thread-safe sam po sebi - sinhronizaciju obezbedjuje BlockManager koji ga koristi.
type BlockCache struct {
	capacity int
	items    map[blockKey]*list.Element
	order    *list.List // Front = najskorije koriscen blok, Back = sledeci kandidat za izbacivanje.
}

// NewBlockCache pravi novi LRU kes kapaciteta capacity blokova.
func NewBlockCache(capacity int) *BlockCache {
	if capacity <= 0 {
		capacity = 1
	}

	return &BlockCache{
		capacity: capacity,
		items:    make(map[blockKey]*list.Element),
		order:    list.New(),
	}
}

// Get vraca kopiju sadrzaja bloka ako se nalazi u kesu i pomera ga na pocetak LRU reda
// (postaje najskorije koriscen). Drugi povratni parametar je false ako bloka nema u kesu.
func (c *BlockCache) Get(path string, blockNumber int) ([]byte, bool) {
	key := blockKey{path: path, blockNumber: blockNumber}

	elem, ok := c.items[key]
	if !ok {
		return nil, false
	}

	c.order.MoveToFront(elem)

	entry := elem.Value.(*cacheEntry)
	result := make([]byte, len(entry.data))
	copy(result, entry.data)

	return result, true
}

// Put ubacuje ili azurira sadrzaj bloka u kesu. Ako je kes pun, izbacuje se
// najdavnije koriscen blok (LRU politika).
func (c *BlockCache) Put(path string, blockNumber int, data []byte) {
	key := blockKey{path: path, blockNumber: blockNumber}

	stored := make([]byte, len(data))
	copy(stored, data)

	if elem, ok := c.items[key]; ok {
		elem.Value.(*cacheEntry).data = stored
		c.order.MoveToFront(elem)
		return
	}

	elem := c.order.PushFront(&cacheEntry{key: key, data: stored})
	c.items[key] = elem

	if c.order.Len() > c.capacity {
		c.evictOldest()
	}
}

// UpdateIfPresent azurira sadrzaj bloka u kesu samo ako je taj blok vec prisutan
// (koristi se prilikom pisanja - spec 1.6 kaze da se pri write operaciji kes azurira
// "ako je blok tu bio prisutan", a ne da se svaki upisan blok nasilno ubacuje u kes).
// Vraca true ako je blok bio prisutan i azuriran je.
func (c *BlockCache) UpdateIfPresent(path string, blockNumber int, data []byte) bool {
	key := blockKey{path: path, blockNumber: blockNumber}

	elem, ok := c.items[key]
	if !ok {
		return false
	}

	stored := make([]byte, len(data))
	copy(stored, data)

	elem.Value.(*cacheEntry).data = stored
	c.order.MoveToFront(elem)

	return true
}

// Invalidate uklanja blok iz kesa ako je u njemu prisutan.
func (c *BlockCache) Invalidate(path string, blockNumber int) {
	key := blockKey{path: path, blockNumber: blockNumber}

	if elem, ok := c.items[key]; ok {
		c.order.Remove(elem)
		delete(c.items, key)
	}
}

// Len vraca trenutni broj blokova u kesu.
func (c *BlockCache) Len() int {
	return c.order.Len()
}

func (c *BlockCache) evictOldest() {
	oldest := c.order.Back()
	if oldest == nil {
		return
	}

	entry := oldest.Value.(*cacheEntry)
	delete(c.items, entry.key)
	c.order.Remove(oldest)
}
