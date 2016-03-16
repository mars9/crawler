package crawler

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	pb "github.com/mars9/crawler/crawlerpb"
)

var errNotFound = errors.New("not found")

// Bucket represents a Record store.
type Bucket interface {
	// Put puts the record for the given key. It overwrites any previous
	// record for that key; a Bucket is not a multi-map.
	Put(key []byte, record *pb.Record) (err error)

	// Get returns the record for the given key. It returns errNotFound if
	// the Bucket does not contain the key.
	Get(key []byte, record *pb.Record) (err error)

	// Delete deletes the record for the given key. It returns errNotFound if
	// the Bucket does not contain the key.
	Delete(key []byte) (err error)

	// Exists returns errNotFound if the Bucket does not contain the key.
	Exists(key []byte) (err error)

	// Bucket returns the name and unique identifier for this Bucket.
	Bucket() (name, uuid []byte)

	// List returns all collected records.
	List() (rec <-chan *pb.Record, err error)
}

// Store implements a boltdb backed record store.
type Store struct {
	db *bolt.DB
}

// NewStore returns a boltdb backed record store.
func NewStore(dbpath string, limit int64) (*Store, error) {
	db, err := bolt.Open(dbpath, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the store, rendering it unusable for I/O.
func (s *Store) Close() error { return s.db.Close() }

// Create creates the named bucket. Create returns an error if it already
// exists. If successful, methods on the returned bucket can be used for
// I/O.
func (s *Store) Create(name []byte) (Bucket, error) {
	now := uint32(time.Now().Unix())
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return nil, err
	}
	uuid := []byte(fmt.Sprintf("%08x-%04x-%04x-%04x-%04x%08x",
		now, b[0:2], b[2:4], b[4:6], b[6:8], b[8:]))

	return createBucket(s.db, [][]byte{name, uuid})
}

// Open opens the named bucket. If successful, methods on the returned
// bucket can be used for I/O.
func (s *Store) Open(name, uuid []byte) (Bucket, error) {
	return openBucket(s.db, name, uuid)
}

func (s *Store) List(name []byte) (<-chan []byte, error) {
	txn, err := s.db.Begin(false)
	if err != nil {
		return nil, err
	}

	bucket := txn.Bucket(name)
	if bucket == nil {
		return nil, bolt.ErrBucketNotFound
	}
	c := bucket.Cursor()

	ch := make(chan []byte, 6)
	go func(ch chan<- []byte, txn *bolt.Tx) {
		defer func() {
			txn.Rollback()
			close(ch)
		}()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			ch <- []byte(k) // This is boltdb, of course we copy it.
		}
	}(ch, txn)

	return ch, nil
}

func (s *Store) ListAll() (<-chan []byte, error) {
	txn, err := s.db.Begin(false)
	if err != nil {
		return nil, err
	}

	c := txn.Cursor()

	ch := make(chan []byte, 6)
	go func(ch chan<- []byte, txn *bolt.Tx) {
		defer func() {
			txn.Rollback()
			close(ch)
		}()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			ch <- []byte(k) // This is boltdb, of course we copy it.
		}
	}(ch, txn)

	return ch, nil
}

// Backup writes the entire database to a writer. A reader transaction is
// maintained during the backup so it is safe to continue using the
// database while a backup is in progress.
func (s *Store) Backup(w io.Writer) (int64, error) {
	txn, err := s.db.Begin(true)
	if err != nil {
		return 0, err
	}
	defer txn.Rollback()

	if err = txn.Copy(w); err != nil {
		return 0, err
	}
	return txn.Size(), nil
}

type boltStore struct {
	db   *bolt.DB
	root [][]byte
}

func createBucket(db *bolt.DB, bucket [][]byte) (*boltStore, error) {
	if len(bucket) == 0 {
		return nil, errors.New("missing root bucket")
	}

	txn, err := db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			txn.Rollback()
		}
	}()

	var b *bolt.Bucket
	if b, err = txn.CreateBucketIfNotExists(bucket[0]); err != nil {
		return nil, err
	}
	if len(bucket) > 1 {
		for i := range bucket[1:] {
			if b, err = b.CreateBucket(bucket[i+1]); err != nil {
				return nil, err
			}
		}
	}
	return &boltStore{db: db, root: bucket}, txn.Commit()
}

func openBucket(db *bolt.DB, name, uuid []byte) (*boltStore, error) {
	txn, err := db.Begin(true)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			txn.Rollback()
		}
	}()

	var b *bolt.Bucket
	if b = txn.Bucket(name); b == nil {
		return nil, bolt.ErrBucketNotFound
	}
	if b = b.Bucket(uuid); b == nil {
		return nil, bolt.ErrBucketNotFound
	}
	return &boltStore{db: db, root: [][]byte{name, uuid}}, txn.Commit()
}

func (s *boltStore) Close() error                { return s.db.Close() }
func (s *boltStore) Bucket() (name, uuid []byte) { return s.root[0], s.root[1] }

func (s *boltStore) findRoot(txn *bolt.Tx) (*bolt.Bucket, error) {
	bucket := txn.Bucket(s.root[0])
	if bucket == nil {
		return nil, errors.New("root bucket not found")
	}

	if len(s.root) > 1 {
		for i := range s.root[1:] {
			bucket = bucket.Bucket(s.root[i+1])
			if bucket == nil {
				return nil, errors.New("root bucket not found")
			}
		}
	}
	return bucket, nil
}

func (s *boltStore) Exists(key []byte) error {
	txn, err := s.db.Begin(false)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	bucket, err := s.findRoot(txn)
	if err != nil {
		return errNotFound
	}
	if data := bucket.Get(key); data == nil {
		return errNotFound
	}
	return nil
}

func (s *boltStore) Put(key []byte, record *pb.Record) error {
	txn, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			txn.Rollback()
		}
	}()

	bucket, err := s.findRoot(txn)
	if err != nil {
		return err
	}
	data, err := proto.Marshal(record)
	if err != nil {
		return err
	}
	if err = bucket.Put(key, data); err != nil {
		return err
	}
	return txn.Commit()
}

func (s *boltStore) Get(key []byte, record *pb.Record) error {
	txn, err := s.db.Begin(false)
	if err != nil {
		return err
	}
	defer txn.Rollback()

	bucket, err := s.findRoot(txn)
	if err != nil {
		return err
	}
	if data := bucket.Get(key); data != nil {
		return proto.Unmarshal(data, record)
	}
	return errNotFound
}

func (s *boltStore) Delete(key []byte) error {
	txn, err := s.db.Begin(true)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			txn.Rollback()
		}
	}()

	bucket, err := s.findRoot(txn)
	if err != nil {
		return err
	}
	if err = bucket.Delete(key); err != nil {
		return err
	}
	return txn.Commit()
}

func (s *boltStore) List() (<-chan *pb.Record, error) {
	txn, err := s.db.Begin(false)
	if err != nil {
		return nil, err
	}

	bucket, err := s.findRoot(txn)
	if err != nil {
		return nil, err
	}

	ch := make(chan *pb.Record)
	go func(ch chan<- *pb.Record, txn *bolt.Tx) {
		defer func() {
			txn.Rollback()
			close(ch)
		}()
		c := bucket.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			rec := &pb.Record{}
			if err = proto.Unmarshal(v, rec); err != nil {
				panic(err)
			}
			ch <- rec
		}
	}(ch, txn)

	return ch, nil
}
