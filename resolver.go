package dnsutils

import (
	"net"
	"sort"

	"github.com/russtone/jobq"
)

// Guard to check that Task implements jobq.Task interface.
var _ jobq.Task = new(Task)

// Task represents DNS resolver task.
type Task struct {

	// Name is a DNS name to resolve.
	Name string

	// Qtypes is a list of Query types to use.
	Qtypes []string

	// Meta is metadata which will be copied to result.
	Meta map[string]interface{}

	// answers is a DNS queries answers (map from Query type to answer for this Query type).
	answers map[string][]string

	// qtypeIdx is a current Query type.
	qtypeIdx int
}

// TaskType allows Task to implement jobq.Task interface.
func (t *Task) TaskType() {}

// done returns true when all required queries for the task are done.
func (t *Task) done() bool {
	return len(t.answers) == len(t.Qtypes)
}

// qtype returns current Query type.
func (t *Task) qtype() string {
	if t.qtypeIdx > len(t.Qtypes) {
		panic("Invalid qtype intex")
	}

	return t.Qtypes[t.qtypeIdx]
}

// setAnswer updates the task answers.
func (t *Task) setAnswer(qtype string, answer []string) {
	t.answers[qtype] = answer
	t.qtypeIdx++
}

// Guard to check that Result implements jobq.Result interface.
var _ jobq.Result = new(Result)

// Result represents DNS resolver task result.
type Result struct {
	// Name is a resolved DNS name.
	Name string

	// Answers is a map from Query type to list of results.
	Answers map[string][]string

	// Meta is a meta data copied from Task.
	Meta map[string]interface{}
}

// ResultType allows Result to implement jobq.Result interface.
func (r *Result) ResultType() {}

// SortAnswers sorts Result answers.
func (r *Result) SortAnswers() {
	for _, a := range r.Answers {
		sort.Strings(a)
	}
}

// IsEmpty returns true if there are no answers in the result.
func (r *Result) IsEmpty() bool {
	count := 0

	for _, ans := range r.Answers {
		count += len(ans)
	}

	return count == 0
}

type Meta map[string]interface{}

// TaskGroup represents group of DNS resolve to process.
type TaskGroup interface {
	// Add adds task to group.
	Add(name string, qtypes []string, meta Meta)

	// Next returns next task Result or error.
	Next(*Result, *error) bool

	// Wait blocks until all tasks in group are processed.
	Wait()

	// Close closes all internal resources.
	// Blocks until all internal goroutines are stopped.
	Close()

	// Speed returns processing speed (tasks per second).
	Speed() float64

	// Progress returns progress as number from 0 to 1.
	Progress() float64
}

// Resolver represents DNS resolve tasks queue.
type Resolver interface {
	TaskGroup

	Start()
	Group() TaskGroup
}

var _ Resolver = new(resolver)

// resolver represents DNS resolver.
type resolver struct {
	pool *Pool

	jobq.Queue
}

// NewResolver creates new resolver instance using provided servers and options.
func NewResolver(servers []net.IP, workersCount, rateLimit int) Resolver {
	r := &resolver{
		pool: NewPool(servers, rateLimit, workersCount),
	}

	r.Queue = jobq.New(r.do, workersCount, workersCount*2)

	return r
}

// Start allows resolver to implement Resolver interface.
func (r *resolver) Start() {
	r.pool.Start()
	r.Queue.Start()
}

// Close allows resolver to implement TaskGroup interface.
func (r *resolver) Close() {
	r.pool.Close()
	r.Queue.Close()
}

func (r *resolver) do(task jobq.Task) (jobq.Result, error) {
	t, ok := task.(*Task)
	if !ok {
		panic("invalid task type")
	}

	server := r.pool.Take()
	qtype := t.qtype()

	answer, err := server.Query(t.Name, qtype)
	if err != nil {
		return nil, jobq.Retry(err)
	}

	t.setAnswer(qtype, answer)

	if !t.done() {
		return nil, jobq.ErrRetry
	}

	res := &Result{
		Name:    t.Name,
		Answers: t.answers,
		Meta:    t.Meta,
	}

	return res, nil
}

// Add allows resolver to implement TaskGroup interface.
func (r *resolver) Add(name string, qtypes []string, meta Meta) {
	add(r.Queue, name, qtypes, meta)
}

// Next allows resolver to implement TaskGroup interface.
func (r *resolver) Next(res *Result, err *error) bool {
	return next(r.Queue, res, err)
}

type group struct {
	jobq.TaskGroup
}

func (g *group) Add(name string, qtypes []string, meta Meta) {
	add(g.TaskGroup, name, qtypes, meta)
}

func (g *group) Next(res *Result, err *error) bool {
	return next(g.TaskGroup, res, err)
}

func (r *resolver) Group() TaskGroup {
	return &group{r.Queue.Group()}
}

func add(q jobq.TaskGroup, name string, qtypes []string, meta Meta) {
	q.Add(&Task{
		Name:     name,
		Qtypes:   qtypes,
		Meta:     meta,
		answers:  make(map[string][]string),
		qtypeIdx: 0,
	})
}

func next(q jobq.TaskGroup, res *Result, err *error) bool {
	var (
		rs jobq.Result
		e  error
	)

	ok := q.Next(&rs, &e)

	if rs != nil {
		*res = *(rs.(*Result))
	}

	if e != nil {
		*err = e
	}

	return ok
}
