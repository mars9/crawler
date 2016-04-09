//go:generate go run internal/gen-error/main.go -o errors.go

package store

import (
	"crypto/sha1"
	"fmt"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

var rawBucket = []byte("__raw__") // raw data blobs storage bucket

type BucketIterator func(*Bucket) bool

type DB struct {
	mu  sync.RWMutex
	err error

	tree *bolt.DB
}

func Open(path string, timeout time.Duration) (*DB, error) {
	tree, err := bolt.Open(path, 0600, &bolt.Options{Timeout: timeout})
	if err != nil {
		return nil, err
	}
	if err = tree.Update(func(tx *bolt.Tx) (err error) {
		_, err = tx.CreateBucketIfNotExists(rawBucket)
		return err
	}); err != nil {
		return nil, err
	}
	return &DB{tree: tree}, nil
}

func (db *DB) Next(name []byte) (*Batch, error) {
	if equals(name, rawBucket) {
		panic(fmt.Sprintf("reserved system name: %s", rawBucket))
	}

	var b *Batch
	err := db.tree.Update(func(tx *bolt.Tx) error {
		root, err := tx.CreateBucketIfNotExists(name)
		if err != nil {
			return err
		}

		key, err := next(root)
		if err != nil {
			return err
		}
		_, err = root.CreateBucket(key[:])
		if err != nil {
			return err
		}
		b = &Batch{
			ctx:   &context{hash: sha1.New()},
			batch: make([][]byte, 0, 128),
			tree:  db.tree,
			name:  ncopy(nil, name),
			key:   key,
		}
		return nil
	})
	return b, checkErr(err)
}

func (db *DB) Open(name []byte, key Key) (*Batch, error) {
	if equals(name, rawBucket) {
		panic(fmt.Sprintf("reserved system name: %s", rawBucket))
	}

	var b *Batch
	err := db.tree.Update(func(tx *bolt.Tx) error {
		if root := tx.Bucket(name); root != nil {
			if root = root.Bucket(key[:]); root != nil {
				b = &Batch{
					ctx:   &context{hash: sha1.New()},
					batch: make([][]byte, 0, 128),
					tree:  db.tree,
					name:  ncopy(nil, name),
					key:   key,
				}
			}
		}
		return nil
	})
	return b, checkErr(err)
}

func (db *DB) Get(name []byte, key Key) (*Bucket, error) {
	if equals(name, rawBucket) {
		panic(fmt.Sprintf("reserved system name: %s", rawBucket))
	}

	var b *Bucket
	err := db.tree.Update(func(tx *bolt.Tx) error {
		if root := tx.Bucket(name); root != nil {
			if root = root.Bucket(key[:]); root != nil {
				b = &Bucket{
					name: ncopy(nil, name),
					key:  key,
					tree: db.tree,
				}
			}
		}
		return nil
	})
	return b, checkErr(err)
}

func (db *DB) Range(iterator BucketIterator) error {
	return nil
}

func (db *DB) Close() error { return db.tree.Close() }

func equals(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(b); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
