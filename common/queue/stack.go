package queue

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"path/filepath"

	log "github.com/Sirupsen/logrus"
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
func NewJobDB(path string) (*JobDB, error) {
	logger := log.WithFields(log.Fields{
		"path": path,
	})
	var jb JobDB

	db, err := bbolt.Open(filepath.Clean(path), 0600, nil)
	if err != nil {
		return &JobDB{}, err
	}
	logger.Debug("Opened boltdb database")

	tx, err := db.Begin(true)
	if err != nil {
		return &JobDB{}, err
	}
	logger.Debug("Begin database setup transaction")

	_, err = tx.CreateBucketIfNotExists(BucketJobs)
	if err != nil {
		tx.Rollback()
		return &JobDB{}, err
	}

	_, err = tx.CreateBucketIfNotExists(BucketJobOrder)
	if err != nil {
		tx.Rollback()
		return &JobDB{}, err
	}

	err = tx.Commit()
	if err != nil {
		return &JobDB{}, err
	}
	logger.Debug("End database setup transaction")

	jb.boltdb = db
	return &jb, nil
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
	logger := log.WithFields(log.Fields{
		"jobID":   j.UUID,
		"jobName": j.Name,
		"params":  common.CleanJobParamsForLogging(j),
	})
	logger.Debug("Attempting to Job to database")

	value, err := json.Marshal(j)
	if err != nil {
		return err
	}
	logger.Debug("Marshaled Job to JSON for storing")

	return db.boltdb.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)
		bo := tx.Bucket(BucketJobOrder)

		err := b.Put([]byte(j.UUID), value)
		if err != nil {
			return err
		}
		id, err := bo.NextSequence()
		if err != nil {
			return err
		}

		err = bo.Put(itob(id), []byte(j.UUID))
		if err != nil {
			return err
		}

		logger.Debug("Job successfully added to the database.")
		return nil
	})
}

// GetJob returns the common.Job for the given UUID
func (db *JobDB) GetJob(uuid string) (common.Job, error) {
	logger := log.WithFields(log.Fields{
		"jobID": uuid,
	})
	logger.Debug("Attempting to get job from the database")

	var j common.Job

	err := db.boltdb.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)

		err := json.Unmarshal(b.Get([]byte(uuid)), &j)
		if err != nil {
			return err
		}

		return nil
	})

	return j, err
}

// GetAllJobs returns the full Queue of Jobs
func (db *JobDB) GetAllJobs() ([]common.Job, error) {
	log.Debug("Attempting to get all jobs from the database")

	var jobs []common.Job

	err := db.boltdb.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)
		bo := tx.Bucket(BucketJobOrder)

		c := bo.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var job common.Job

			err := json.Unmarshal(b.Get(v), &job)
			if err != nil {
				return err
			}

			jobs = append(jobs, job)
		}

		return nil
	})

	logger := log.WithFields(log.Fields{
		"jobCount": len(jobs),
	})
	logger.Debug("Walked database collecting jobs")
	return jobs, err
}

// UpdateJob updates the value of a common.Job already in the database
func (db *JobDB) UpdateJob(j common.Job) error {
	logger := log.WithFields(log.Fields{
		"uuid":   j.UUID,
		"name":   j.Name,
		"params": common.CleanJobParamsForLogging(j),
	})
	logger.Debug("Attempting to update job in the database")

	value, err := json.Marshal(j)
	if err != nil {
		return err
	}
	logger.Debug("Job marshaled to JSON")

	return db.boltdb.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)

		return b.Put([]byte(j.UUID), value)
	})
}

// DeleteJob removes a given common.Job by UUID from the DB
func (db *JobDB) DeleteJob(uuid string) error {
	logger := log.WithFields(log.Fields{
		"jobID": uuid,
	})
	logger.Debug("Attempting to delete job in the database")

	return db.boltdb.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(BucketJobs)
		bo := tx.Bucket(BucketJobOrder)

		logger.Debug("Find job in order bucket for removal")
		var boKey []byte
		c := bo.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if bytes.Equal(v, []byte(uuid)) {
				boKey = k
			}
		}

		boLogger := log.WithField("orderKey", boKey)
		boLogger.Debug("Order key found, now delete the value")
		err := bo.Delete(boKey)
		if err != nil {
			return err
		}

		logger.Debug("Deleted job value based on UUID")
		return b.Delete([]byte(uuid))
	})
}

// ReorderJobs deletes the order bucket, recreates it with new order
func (db *JobDB) ReorderJobs(uuids []string) error {
	log.Debug("Attempting to reorder jobs in the database")

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
			id, err := bo.NextSequence()
			if err != nil {
				return err
			}
			err = bo.Put(itob(id), []byte(uuids[i]))
			if err != nil {
				return err
			}
		}

		return nil
	})
}
