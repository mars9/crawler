// generated by gen-error -o errors.go; DO NOT EDIT

package store

import "github.com/boltdb/bolt"

type StorageError string

func (e StorageError) Error() string { return string(e) }

type Error string

func (e Error) Error() string { return string(e) }

func checkErr(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case err == bolt.ErrBucketExists:
		return StorageError(err.Error())
	case err == bolt.ErrDatabaseOpen:
		return Error(err.Error())
	case err == bolt.ErrVersionMismatch:
		return StorageError(err.Error())
	case err == bolt.ErrChecksum:
		return StorageError(err.Error())
	case err == bolt.ErrBucketNotFound:
		return StorageError(err.Error())
	case err == bolt.ErrTxClosed:
		return StorageError(err.Error())
	case err == bolt.ErrInvalid:
		return StorageError(err.Error())
	case err == bolt.ErrTxNotWritable:
		return StorageError(err.Error())
	case err == bolt.ErrBucketNameRequired:
		return StorageError(err.Error())
	case err == bolt.ErrKeyTooLarge:
		return Error(err.Error())
	case err == bolt.ErrValueTooLarge:
		return Error(err.Error())
	case err == bolt.ErrIncompatibleValue:
		return StorageError(err.Error())
	case err == bolt.ErrDatabaseNotOpen:
		return Error(err.Error())
	case err == bolt.ErrTimeout:
		return Error(err.Error())
	case err == bolt.ErrDatabaseReadOnly:
		return StorageError(err.Error())
	case err == bolt.ErrKeyRequired:
		return Error(err.Error())
	}
	return err // return original error
}