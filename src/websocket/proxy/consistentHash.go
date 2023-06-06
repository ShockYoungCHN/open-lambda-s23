package proxy

import (
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/emirpasic/gods/utils"
	"hash/crc32"
	"hash/fnv"
	"strconv"
	"sync"
)

type ConsistentHash struct {
	circle    *treemap.Map      // uint32->*vnode
	ipToVnode map[string]*vnode // ip->*vnode, used for removing

	count   int      // total number of virtual nodes
	scratch [64]byte // scratch space for hashing
	UseFnv  bool     // use FNV-1a hash algorithm or CRC32
	sync.RWMutex
}

type vnode struct {
	server string
	count  int // number of virtual nodes assigned to one server
}

func NewConsistentHash() *ConsistentHash {
	m := treemap.NewWith(utils.UInt32Comparator)
	return &ConsistentHash{
		m,
		make(map[string]*vnode),
		0,
		[64]byte{},
		true,
		sync.RWMutex{},
	}
}

func (c *ConsistentHash) Add(server string) {
	c.Lock()
	defer c.Unlock()

	// todo: dynamically adjust 'count'
	c.add(server, 10)
}

func (c *ConsistentHash) add(server string, count int) {
	c.count += count
	for i := 0; i < count; i++ {
		c.circle.Put(c.hashKey(c.indexedKey(server, i)),
			&vnode{server, count})
	}
}

// Remove removes an element from the hash.
func (c *ConsistentHash) Remove(server string) {
	c.Lock()
	defer c.Unlock()
	c.remove(server)
}

func (c *ConsistentHash) remove(server string) {
	vnode, _ := c.ipToVnode[server]
	for i := 0; i < vnode.count; i++ {
		c.circle.Remove(c.hashKey(c.indexedKey(server, i)))
	}
	c.count--
}

// Get returns the server that a given key is assigned to.
func (c *ConsistentHash) Get(s string) string {
	c.RLock()
	defer c.RUnlock()
	key := c.hashKey(s)
	hash, addr := c.circle.Ceiling(key)
	if hash == nil {
		hash, addr = c.circle.Min()
	}
	return addr.(string)
}

// hash utils
func (c *ConsistentHash) hashKey(key string) uint32 {
	if c.UseFnv {
		return c.hashKeyFnv(key)
	}
	return c.hashKeyCRC32(key)
}

func (c *ConsistentHash) hashKeyCRC32(key string) uint32 {
	if len(key) < 64 {
		var scratch [64]byte
		copy(scratch[:], key)
		return crc32.ChecksumIEEE(scratch[:len(key)])
	}
	return crc32.ChecksumIEEE([]byte(key))
}

func (c *ConsistentHash) hashKeyFnv(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// indexedKey generate
func (c *ConsistentHash) indexedKey(elt string, idx int) string {
	// return elt + "|" + strconv.Itoa(idx)
	return strconv.Itoa(idx) + elt
}
