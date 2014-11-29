package crawler

import (
	"os"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/golang/protobuf/proto"
	pb "github.com/mars9/crawler/crawlerpb"
)

func TestBoltStore(t *testing.T) {
	t.Parallel()

	db, err := bolt.Open("test-bolt-store.db", 0644, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = db.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
		os.RemoveAll("test-bolt-store.db")
	}()

	s, err := createBucket(db, [][]byte{[]byte("test-bucket"), []byte("x")})
	if err != nil {
		t.Fatalf("new bolt store: %v", err)
	}

	_, err = createBucket(db, [][]byte{[]byte("test-bucket"), []byte("x")})
	assert(t, "error", bolt.ErrBucketExists, err)
	_, err = createBucket(db, [][]byte{[]byte("test-bucket"), []byte("y")})
	assert(t, "error", nil, err)
	_, err = createBucket(db, [][]byte{[]byte("test-bucket"), []byte("y")})
	assert(t, "error", bolt.ErrBucketExists, err)

	rec1, rec2 := pb.Record{
		URL:   proto.String("http://example.com"),
		Key:   []byte("unique record key"),
		Title: proto.String("new site"),
	}, pb.Record{}

	key := []byte("test-key")
	if err = s.Put(key, &rec1); err != nil {
		t.Fatal("put record: %v", err)
	}

	assert(t, "exists error", nil, s.Exists(key))
	assert(t, "exists error", errNotFound, s.Exists([]byte("unknown")))

	if err = s.Get(key, &rec2); err != nil {
		t.Fatalf("get record: %v", err)
	}
	assert(t, "record", rec1, rec2)

	assert(t, "delete error", nil, s.Delete(key))
	assert(t, "delete error", nil, s.Delete(key)) // delete not existing key/value
	assert(t, "delete error", nil, s.Delete([]byte("unknown")))

	assert(t, "get error", errNotFound, s.Get(key, nil))
}

func TestStore(t *testing.T) {
	t.Parallel()

	store, err := NewStore("test-store.db", 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err = store.Close(); err != nil {
			t.Fatalf("close db: %v", err)
		}
		os.RemoveAll("test-store.db")
	}()

	names := [][]byte{
		[]byte("crawler1"),
		[]byte("crawler2"),
		[]byte("crawler3"),
		[]byte("crawler4"),
	}
	var bucket Bucket
	for i, name := range names {
		if i == 0 {
			if bucket, err = store.Create(name); err != nil {
				t.Fatalf("create %s: %v", name, err)
			}
			continue
		}
		if _, err = store.Create(name); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	alistc, err := store.ListAll()
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	var got [][]byte
	for b := range alistc {
		got = append(got, b)
	}
	assert(t, "crawlers", names, got)

	listc, err := store.List(names[0])
	if err != nil {
		t.Fatalf("list %s: %v", names[0], err)
	}
	var uuid [][]byte
	for b := range listc {
		uuid = append(uuid, b)
	}
	assert(t, "length", 1, len(uuid))

	_, wantUUID := bucket.Bucket()
	assert(t, "uuid", wantUUID, uuid[0])

	nbucket, err := store.Open(names[0], uuid[0])
	if err != nil {
		t.Fatalf("open %s %s: %v", names[0], uuid[0])
	}
	assert(t, "bucket", bucket.(*boltStore), nbucket.(*boltStore))
}
