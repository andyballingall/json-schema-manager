package schema

import (
	"io"
	"sync"
	"time"
)

// Reporter defines the interface for creating formatted test reports.
type Reporter interface {
	Write(w io.Writer, report *TestReport) error
}

// TestLog is a map of schema keys to a list of Specs.
// It is used to collect test outcomes for a schema identified by Key.
type TestLog map[Key][]Spec

func NewTestLog() TestLog {
	return make(TestLog)
}

func (l TestLog) AddTest(key Key, spec *Spec) {
	l[key] = append(l[key], *spec)
}

type TestReport struct {
	mu sync.Mutex

	StartTime time.Time
	EndTime   time.Time
	// Fields for future implementation when test execution is added
	// numTests    int     // The number of tests that were run
	// numFailures int     // The number of tests which identified a problem.
	FailedTests TestLog // tests exposing a problem - i.e. a pass test that failed or a fail test that passed
	PassedTests TestLog // tests that passed as expected - i.e a pass test that passed and a fail test that failed
}

func NewTestReport() *TestReport {
	return &TestReport{
		FailedTests: make(TestLog),
		PassedTests: make(TestLog),
	}
}

func (r *TestReport) AddFailedTest(key Key, spec *Spec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.FailedTests[key] = append(r.FailedTests[key], *spec)
}

func (r *TestReport) AddPassedTest(key Key, spec *Spec) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.PassedTests[key] = append(r.PassedTests[key], *spec)
}
