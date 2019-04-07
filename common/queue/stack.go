package queue

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"path/filepath"

	"github.com/etcd-io/bbolt"
	"github.com/jmmcatee/cracklord/common"
)

var BucketJobs = []byte("jobs")
var BucketJobOrder = []byte("jobs-order")

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// JobDB is a structure to store and process jobs for the Queue
type JobDB struct {
	boltdb *bbolt.DB
}

// NewJobDB returns a new instance of the JobDB structure
func NewJobDB(path string) (JobDB, error) {
	var jb JobDB

	db, err := bbolt.Open(filepath.Clean(path), 0600, nil)
	if err != nil {
		return JobDB{}, err
	}

	tx, err := db.Begin(true)
	if err != nil {
		return JobDB{}, err
	}

	_, err = tx.CreateBucketIfNotExists(BucketJobs)
	if err != nil {
		tx.Rollback()
		return JobDB{}, err
	}

	_, err = tx.CreateBucketIfNotExists(BucketJobOrder)
	if err != nil {
		tx.Rollback()
		return JobDB{}, err
	}

	jb.boltdb = db
	return jb, nil
}

// Count returns the number of jobs in the queue
func (db *JobDB) Count() int {
	var count int
	db.boltdb.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)

		count = b.Stats().KeyN

		return nil
	})

	return count
}

// AddJob adds a common.Job structure to the BBoltDB
func (db *JobDB) AddJob(j common.Job) error {
	value, err := json.Marshal(j)
	if err != nil {
		return err
	}

	return db.boltdb.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)
		bo := tx.Bucket(BucketJobOrder)

		b.Put([]byte(j.UUID), value)
		id, _ := bo.NextSequence()
		bo.Put(itob(id), []byte(j.UUID))

		return nil
	})
}

// GetJob returns the common.Job for the given UUID
func (db *JobDB) GetJob(uuid string) (common.Job, error) {
	var j common.Job

	err := db.boltdb.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)

		value := b.Get([]byte(uuid))

		err := json.Unmarshal(value, &j)
		if err != nil {
			return err
		}

		return nil
	})

	return j, err
}

// GetAllJobs returns the full Queue of Jobs
func (db *JobDB) GetAllJobs() ([]common.Job, error) {
	var jobs []common.Job
	var job common.Job

	err := db.boltdb.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)
		bo := tx.Bucket(BucketJobOrder)

		c := bo.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			jvalue := b.Get(v)
			err := json.Unmarshal(jvalue, &job)
			if err != nil {
				return err
			}

			jobs = append(jobs, job)
		}

		return nil
	})

	return jobs, err
}

// UpdateJob updates the value of a common.Job already in the database
func (db *JobDB) UpdateJob(j common.Job) error {
	value, err := json.Marshal(j)
	if err != nil {
		return err
	}

	return db.boltdb.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)

		return b.Put([]byte(j.UUID), value)
	})
}

// DeleteJob removes a given common.Job by UUID from the DB
func (db *JobDB) DeleteJob(uuid string) error {
	return db.boltdb.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)
		bo := tx.Bucket(BucketJobOrder)

		var boKey []byte
		c := bo.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if bytes.Equal(v, []byte(uuid)) {
				boKey = k
			}
		}
		err := bo.Delete(boKey)
		if err != nil {
			return err
		}

		return b.Delete([]byte(uuid))
	})
}

// ReorderJobs deletes the order bucket, recreates it with new order
func (db *JobDB) ReorderJobs(uuids []string) error {
	return db.boltdb.Update(func(tx *bbolt.Tx) error {
		bo := tx.Bucket(BucketJobOrder)
		c := bo.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			// Check if this key is in the order we got
			var found bool
			for i := range uuids {
				if uuids[i] == string(v) {
					found = true
				}
			}

			if !found {
				uuids = append(uuids, string(v))
			}
		}

		err := tx.DeleteBucket(BucketJobOrder)
		if err != nil {
			return err
		}

		bo, err = tx.CreateBucket(BucketJobOrder)
		if err != nil {
			return err
		}

		for i := range uuids {
			id, _ := bo.NextSequence()
			bo.Put(itob(id), []byte(uuids[i]))
		}

		return nil
	})
}
