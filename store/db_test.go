package store

import (
	"fmt"
	"os"
	"testing"
)

var testBatch [][]byte

func init() {
	for i := 1; i <= 1000; i++ {
		value := []byte(fmt.Sprintf("value#%.16d", i))
		testBatch = append(testBatch, value)
	}
}

func basicTest(t *testing.T, db *DB) {
	bucket := []byte("basic_test_bucket")
	batch, err := db.Next(bucket)
	if err != nil {
		t.Fatalf("creating bucket: %v", err)
	}

	_, err = batch.Append(testBatch[:15], true)
	if err != nil {
		t.Fatalf("append batch: %v", err)
	}
	_, err = batch.Append(testBatch[15:30], true)
	if err != nil {
		t.Fatalf("append batch: %v", err)
	}

	root, err := db.Get(bucket, batch.Key())
	if err != nil {
		t.Fatalf("get bucket: %v", err)
	}

	fmt.Printf("%q - %q\n", root.Name(), root.Key())
	err = root.Range(func(key Key, val []byte) bool {
		fmt.Printf("\t%q - %q\n", key, val)
		return true
	})
	if err != nil {
		t.Fatalf("bucket iterator: %v", err)
	}
}

func openTestDB(t *testing.T, name string) (*DB, string) {
	db, err := Open(name+"___test___database___.db", 0)
	if err != nil {
		t.Fatalf("cannot open database: %v", err)
	}
	return db, name + "___test___database___.db"
}

func closeTestDB(t *testing.T, db *DB, name string) {
	if err := db.Close(); err != nil {
		t.Fatalf("closint database: %v", err)
	}
	os.Remove(name)
}

func TestDB(t *testing.T) {
	db, name := openTestDB(t, "")
	defer closeTestDB(t, db, name)

	basicTest(t, db)
}
