package store

import (
	"encoding/binary"
	"fmt"
	"hash"
	"sync"

	"github.com/boltdb/bolt"
)

var first = Key{'\x00', '\x00', '\x00', '\x00', '\x00', '\x00', '\x00', '\x01'}

type Key [8]byte

func (k Key) String() string { return fmt.Sprintf("0x%x", k[:]) }

type context struct {
	root *bolt.Bucket
	raw  *bolt.Bucket
	hash hash.Hash
}

func newReadonly(tx *bolt.Tx, name, key []byte) *context {
	ctx := &context{} // readonly context do not use hash
	ctx.init(tx, name, key)
	return ctx
}

func (c *context) init(tx *bolt.Tx, name, key []byte) {
	c.root = tx.Bucket(name).Bucket(key)
	c.raw = tx.Bucket(rawBucket)
}

func (c *context) free() { c.root, c.raw = nil, nil }

func (c *context) iterate(prefix []byte, iterator RecordIterator) error {
	root := c.root.Cursor()
	var key, hash []byte
	if prefix != nil {
		key, hash = root.Seek(prefix)
	} else {
		key, hash = root.First()
	}
	for ; key != nil; key, hash = root.Next() {
		if val := c.raw.Get(hash); val != nil {
			value, err := uncompress(val)
			if err != nil {
				return err
			}
			if !iterator(kcopy(key), value) {
				return nil
			}
			continue
		}
		panic(fmt.Sprintf("score 0x%x not found", hash))
	}
	return nil
}

func (c *context) get(key []byte) ([]byte, error) {
	if hash := c.root.Get(key); hash != nil {
		if val := c.raw.Get(hash); val != nil {
			return uncompress(val)
		}
		panic(fmt.Sprintf("score 0x%x not found", hash))
	}
	return nil, nil
}

func (c *context) append(batch [][]byte) (n int, err error) {
	for _, value := range batch {
		hash := checksum(c.hash, value)
		value = compress(value)
		if err = c.compareAndSwap(hash[:], value); err != nil {
			return n, err
		}

		key, err := next(c.root)
		if err != nil {
			return n, err
		}
		if err = c.root.Put(key[:], hash[:20]); err != nil {
			return n, err
		}
		n++
	}
	return n, err
}

func (c *context) compareAndSwap(hash, value []byte) error {
	if val := c.raw.Get(hash[:20]); val != nil {
		ref := binary.BigEndian.Uint64(c.raw.Get(hash[:]))
		ref++
		var refCount [8]byte
		binary.BigEndian.PutUint64(refCount[:], ref)
		return c.raw.Put(hash[:], refCount[:])
	}

	if err := c.raw.Put(hash[:], first[:]); err != nil {
		return err
	}
	return c.raw.Put(hash[:20], value)
}

func checksum(hash hash.Hash, value []byte) (sum [22]byte) {
	hash.Write(value)
	hash.Sum(sum[:0])
	sum[20] = '-'
	sum[21] = 'x'
	hash.Reset()
	return sum
}

func next(b *bolt.Bucket) (key Key, err error) {
	seq, err := b.NextSequence()
	if err != nil {
		return key, err
	}
	binary.BigEndian.PutUint64(key[:], seq)
	return key, err
}

type Batch struct {
	mu    sync.Mutex
	batch [][]byte
	ctx   *context

	tree *bolt.DB
	name []byte
	key  Key
}

func (b *Batch) append(size int, force bool) (n int, err error) {
	if force || len(b.batch) > size {
		err = b.tree.Update(func(tx *bolt.Tx) error {
			b.ctx.init(tx, b.name, b.key[:])
			n, err = b.ctx.append(b.batch)
			b.batch = b.batch[:0]
			b.ctx.free()
			return err
		})
	}
	return n, err
}

func (b *Batch) flush(ctx *context) (n int, err error) {
	if len(b.batch) > 0 {
		n, err = b.ctx.append(b.batch)
		b.batch = b.batch[:0]
	}
	return n, err
}

func (b *Batch) Append(batch [][]byte, sync bool) (n int, err error) {
	b.mu.Lock()
	b.batch = append(b.batch, batch...)
	n, err = b.append(128, sync)
	b.mu.Unlock()
	return n, checkErr(err)
}

func (b *Batch) Put(value []byte, sync bool) (n int, err error) {
	b.mu.Lock()
	b.batch = append(b.batch, value)
	n, err = b.append(128, sync)
	b.mu.Unlock()
	return n, err
}

func (b *Batch) Commit() (n int, err error) {
	b.mu.Lock()
	n, err = b.append(0, false)
	b.mu.Unlock()
	return n, checkErr(err)
}

func (b *Batch) Delete(key []byte) (err error) {
	b.mu.Lock()
	err = b.tree.Update(func(tx *bolt.Tx) error {
		b.ctx.init(tx, b.name, b.key[:])
		defer b.ctx.free()

		if _, err = b.flush(b.ctx); err != nil {
			return err
		}
		return b.ctx.root.Delete(key)
	})
	b.mu.Unlock()
	return checkErr(err)
}

func (b *Batch) Remove() (err error) {
	b.mu.Lock()
	err = b.tree.Update(func(tx *bolt.Tx) error {
		err = tx.Bucket(b.name).DeleteBucket(b.key[:])
		b.batch = nil
		return err
	})
	b.mu.Unlock()
	return checkErr(err)
}

func (b *Batch) From(key []byte, iterator RecordIterator) error {
	b.mu.Lock()
	err := b.tree.View(func(tx *bolt.Tx) error {
		b.ctx.init(tx, b.name, b.key[:])
		if _, err := b.flush(b.ctx); err != nil {
			return err
		}
		err := b.ctx.iterate(key, iterator)
		b.ctx.free()
		return err
	})
	b.mu.Unlock()
	return checkErr(err)
}

func (b *Batch) Range(iterator RecordIterator) error {
	b.mu.Lock()
	err := b.tree.View(func(tx *bolt.Tx) error {
		b.ctx.init(tx, b.name, b.key[:])
		if _, err := b.flush(b.ctx); err != nil {
			return err
		}
		err := b.ctx.iterate(nil, iterator)
		b.ctx.free()
		return err
	})
	b.mu.Unlock()
	return checkErr(err)
}

func (b *Batch) Get(key []byte) (value []byte, err error) {
	b.mu.Lock()
	err = b.tree.View(func(tx *bolt.Tx) error {
		b.ctx.init(tx, b.name, b.key[:])
		if _, err = b.flush(b.ctx); err != nil {
			return err
		}
		value, err = b.ctx.get(key)
		b.ctx.free()
		return err
	})
	b.mu.Unlock()
	return value, checkErr(err)
}

func (b *Batch) Name() []byte { return b.name }
func (b *Batch) Key() Key     { return b.key }

type RecordIterator func(Key, []byte) bool

type Bucket struct {
	tree *bolt.DB
	name []byte
	key  Key
}

func (b *Bucket) From(key []byte, iterator RecordIterator) error {
	return checkErr(b.tree.View(func(tx *bolt.Tx) error {
		ctx := newReadonly(tx, b.name, b.key[:])
		err := ctx.iterate(key, iterator)
		ctx.free()
		return err
	}))
}

func (b *Bucket) Range(iterator RecordIterator) error {
	return checkErr(b.tree.View(func(tx *bolt.Tx) error {
		ctx := newReadonly(tx, b.name, b.key[:])
		err := ctx.iterate(nil, iterator)
		ctx.free()
		return err
	}))
}

func (b *Bucket) Get(key []byte) (value []byte, err error) {
	err = checkErr(b.tree.View(func(tx *bolt.Tx) error {
		ctx := newReadonly(tx, b.name, b.key[:])
		value, err = ctx.get(key)
		ctx.free()
		return err
	}))
	return value, err
}

func (b *Bucket) Name() []byte { return b.name }
func (b *Bucket) Key() Key     { return b.key }

func ncopy(dst, src []byte) []byte {
	n := len(src)
	if cap(dst) < n {
		dst = make([]byte, n)
	}
	dst = dst[:n]
	copy(dst, src)
	return dst
}

func kcopy(key []byte) Key {
	var k Key
	copy(k[:], key)
	return k
}
