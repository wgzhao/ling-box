package launcher

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Job describes one background tool invocation.
type Job struct {
	Tool    string   `json:"tool"`
	Argv    []string `json:"argv"`
	Stdin   string   `json:"stdin,omitempty"`
	Timeout int      `json:"timeout_seconds"`
}

// JobResult is what the worker writes and the launcher reads back.
type JobResult struct {
	OK     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// JobStore keeps background job state in the user's temp directory. The
// Script Filter is invoked repeatedly by Alfred's rerun, so state has to
// outlive a single process.
type JobStore struct {
	Dir string
}

// NewJobStore returns the default store.
func NewJobStore() JobStore {
	return JobStore{Dir: filepath.Join(os.TempDir(), "lingbox-launcher")}
}

// Key returns a stable, filesystem-safe key for a query.
func (s JobStore) Key(query string) string {
	sum := sha256.Sum256([]byte(query))
	return hex.EncodeToString(sum[:8])
}

func (s JobStore) paths(key string) (job, result, lock string) {
	return filepath.Join(s.Dir, key+".job"),
		filepath.Join(s.Dir, key+".json"),
		filepath.Join(s.Dir, key+".lock")
}

// Load returns a completed result, if any.
func (s JobStore) Load(key string) (JobResult, bool) {
	_, resultPath, _ := s.paths(key)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		return JobResult{}, false
	}
	var res JobResult
	if err := json.Unmarshal(data, &res); err != nil {
		return JobResult{}, false
	}
	return res, true
}

// LockAge reports how long the job has been running.
func (s JobStore) LockAge(key string) (time.Duration, bool) {
	_, _, lockPath := s.paths(key)
	fi, err := os.Stat(lockPath)
	if err != nil {
		return 0, false
	}
	return time.Since(fi.ModTime()), true
}

// Begin reserves the job and writes its description. It fails if another
// invocation already holds the reservation.
func (s JobStore) Begin(key string, job Job) (string, error) {
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return "", err
	}
	jobPath, _, lockPath := s.paths(key)
	lock, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", err
	}
	_ = lock.Close()

	data, err := json.Marshal(job)
	if err != nil {
		s.Abort(key)
		return "", err
	}
	if err := writeAtomic(jobPath, data); err != nil {
		s.Abort(key)
		return "", err
	}
	return jobPath, nil
}

// ReadJob loads a job description for the worker.
func ReadJob(jobPath string) (Job, error) {
	data, err := os.ReadFile(jobPath)
	if err != nil {
		return Job{}, err
	}
	var job Job
	if err := json.Unmarshal(data, &job); err != nil {
		return Job{}, err
	}
	return job, nil
}

// CompleteJob writes the worker's result next to the job file and releases
// the reservation. The write is atomic, so a polling Script Filter never
// sees a half-written result.
func CompleteJob(jobPath string, res JobResult) error {
	key := strings.TrimSuffix(filepath.Base(jobPath), ".job")
	store := JobStore{Dir: filepath.Dir(jobPath)}
	data, err := json.Marshal(res)
	if err != nil {
		return err
	}
	_, resultPath, lockPath := store.paths(key)
	if err := writeAtomic(resultPath, data); err != nil {
		return err
	}
	_ = os.Remove(jobPath)
	_ = os.Remove(lockPath)
	return nil
}

// Remove deletes all state for a key.
func (s JobStore) Remove(key string) {
	jobPath, resultPath, lockPath := s.paths(key)
	_ = os.Remove(jobPath)
	_ = os.Remove(resultPath)
	_ = os.Remove(lockPath)
}

// Abort releases a reservation without producing a result.
func (s JobStore) Abort(key string) {
	jobPath, _, lockPath := s.paths(key)
	_ = os.Remove(jobPath)
	_ = os.Remove(lockPath)
}

// Cleanup removes state older than maxAge. It is best-effort and safe to
// call on every Script Filter invocation.
func (s JobStore) Cleanup(maxAge time.Duration) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || info.ModTime().After(cutoff) {
			continue
		}
		_ = os.Remove(filepath.Join(s.Dir, e.Name()))
	}
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
