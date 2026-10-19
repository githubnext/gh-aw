package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

const maxRecords = 10000

type quarantineError struct {
	reason string
}

func (e quarantineError) Error() string { return e.reason }

type Job struct {
	Route   string    `json:"route"`
	Event   string    `json:"event"`
	Payload Payload   `json:"payload"`
	State   string    `json:"state"`
	Attempt int       `json:"attempt"`
	Next    time.Time `json:"next"`
	Error   string    `json:"error,omitempty"`
}

type Record struct {
	SourceID      string `json:"source_id"`
	DeliveryID    string `json:"delivery_id"`
	BodySHA256    string `json:"body_sha256"`
	Discriminator string `json:"discriminator"`
	Approval      string `json:"approval"`
	Jobs          []Job  `json:"jobs"`
}

type Store struct {
	mu      sync.Mutex
	worker  sync.Mutex
	path    string
	lock    *os.File
	records map[string]Record
}

func openStore(path string) (_ *Store, resultErr error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, errors.New("cannot create durable state directory")
	}
	lock, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, errors.New("cannot open state lock")
	}
	transferred := false
	defer func() {
		if !transferred {
			resultErr = errors.Join(resultErr, lock.Close())
		}
	}()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return nil, errors.New("state already owned by another process")
	}
	s := &Store{path: path, lock: lock, records: map[string]Record{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		transferred = true
		return s, nil
	}
	if err != nil || parseJSON(data, &s.records) != nil || s.records == nil {
		return nil, errors.New("durable state is unreadable or corrupt")
	}
	recovered := s.snapshot()
	for key, record := range recovered {
		for i, job := range record.Jobs {
			if job.State == "sending" {
				job.State, job.Error = "quarantined", "process stopped during an external operation; outcome requires reconciliation"
				record.Jobs[i] = job
			}
		}
		recovered[key] = record
	}
	if err := s.save(recovered); err != nil {
		return nil, err
	}
	transferred = true
	return s, nil
}

func (s *Store) close() error {
	return s.lock.Close()
}

func (s *Store) save(records map[string]Record) (resultErr error) {
	data, err := json.Marshal(records)
	if err != nil {
		return errors.New("cannot serialize durable state")
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), ".gateway-state-*")
	if err != nil {
		return errors.New("cannot create durable state transaction")
	}
	name := f.Name()
	defer func() {
		if f != nil {
			resultErr = errors.Join(resultErr, f.Close())
		}
		if err := os.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			resultErr = errors.Join(resultErr, errors.New("cannot remove temporary state file"))
		}
	}()
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	f = nil
	if err != nil || closeErr != nil || os.Rename(name, s.path) != nil {
		return errors.New("cannot commit durable state")
	}
	dir, err := os.Open(filepath.Dir(s.path))
	if err != nil {
		return errors.New("cannot sync durable state directory")
	}
	defer func() { resultErr = errors.Join(resultErr, dir.Close()) }()
	syncErr := dir.Sync()
	if syncErr != nil {
		return errors.New("cannot sync durable state directory")
	}
	s.records = records
	return nil
}

func (s *Store) snapshot() map[string]Record {
	result := make(map[string]Record, len(s.records))
	for key, record := range s.records {
		record.Jobs = append([]Job(nil), record.Jobs...)
		result[key] = record
	}
	return result
}

func (s *Store) readRecords() map[string]Record {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshot()
}

func (s *Store) saveRecord(key string, record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	records := s.snapshot()
	records[key] = record
	return s.save(records)
}

func (s *Store) admit(record Record) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := record.SourceID + "\x00" + record.DeliveryID
	if prior, ok := s.records[key]; ok {
		if prior.BodySHA256 != record.BodySHA256 || prior.Discriminator != record.Discriminator {
			return false, errors.New("conflicting original delivery identity")
		}
		return false, nil
	}
	// Provider delivery headers are not necessarily signed. Suppress byte-for-byte
	// replay even when a caller changes the delivery header.
	for _, prior := range s.records {
		if prior.SourceID == record.SourceID && prior.BodySHA256 == record.BodySHA256 {
			if prior.Discriminator != record.Discriminator {
				return false, errors.New("conflicting discriminator for authenticated original content")
			}
			return false, nil
		}
	}
	if len(s.records) >= maxRecords {
		return false, errors.New("durable admission capacity exhausted; no replay records were evicted")
	}
	records := s.snapshot()
	records[key] = record
	if err := s.save(records); err != nil {
		return false, err
	}
	return true, nil
}

// A single worker owns dispatch. Durable pending records survive restart. An
// ambiguous network result is terminal rather than risking a duplicate dispatch.
func (s *Store) process(now time.Time, dispatch func(Record, Job) (bool, error)) error {
	s.worker.Lock()
	defer s.worker.Unlock()
	snapshot := s.readRecords()
	for key, record := range snapshot {
		for i, job := range record.Jobs {
			if job.State != "pending" || now.Before(job.Next) {
				continue
			}
			job.State, job.Attempt = "sending", job.Attempt+1
			record.Jobs[i] = job
			if err := s.saveRecord(key, record); err != nil {
				return err
			}
			retry, dispatchErr := dispatch(record, job)
			job.State, job.Error = "sent", ""
			if dispatchErr != nil {
				job.State, job.Error = "failed", dispatchErr.Error()
				var uncertain quarantineError
				if errors.As(dispatchErr, &uncertain) {
					job.State = "quarantined"
				}
				if retry && job.Attempt < 5 {
					job.State = "pending"
					job.Next = now.Add(time.Second * time.Duration(1<<job.Attempt))
				}
			}
			record.Jobs[i] = job
			if err := s.saveRecord(key, record); err != nil {
				return err
			}
		}
	}
	return nil
}
