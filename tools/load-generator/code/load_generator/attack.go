package loadgenerator

import (
	"bytes"
	"crypto/tls"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"
	"unsafe"

	"github.com/valyala/fasthttp"
	"go.uber.org/atomic"
	"gvisor.dev/gvisor/pkg/gohacks"
)

// Attacker is an attack executor which wraps an http.Client
type Attacker struct {
	stopch  chan struct{}
	workers uint64
	maxBody int64
	seq     atomic.Uint64
	began   time.Time
	chunked bool
	timeout time.Duration

	metricFactory func() (*Metrics, error)
}

const (
	// DefaultRedirects is the default number of times an Attacker follows
	// redirects.
	DefaultRedirects = 10
	// DefaultTimeout is the default amount of time an Attacker waits for a request
	// before it times out.
	DefaultTimeout = 30 * time.Second
	// DefaultConnections is the default amount of max open idle connections per
	// target host.
	DefaultConnections = 10000
	// DefaultMaxConnections is the default amount of connections per target
	// host.
	DefaultMaxConnections = 0
	// DefaultWorkers is the default initial number of workers used to carry an attack.
	DefaultWorkers = 10
	// DefaultMaxWorkers is the default maximum number of workers used to carry an attack.
	DefaultMaxWorkers = math.MaxUint64
	// DefaultMaxBody is the default max number of bytes to be read from response bodies.
	// Defaults to no limit.
	DefaultMaxBody = int64(-1)
	// NoFollow is the value when redirects are not followed but marked successful
	NoFollow = -1
)

var (
	// DefaultTLSConfig is the default tls.Config an Attacker uses.
	//nolint:gosec // No sensitive data is sent during performance tests.
	DefaultTLSConfig = &tls.Config{InsecureSkipVerify: true}
)

// NewAttacker returns a new Attacker with default options which are overridden
// by the optionally provided opts.
func NewAttacker(metricFactory func() (*Metrics, error), opts ...func(*Attacker)) *Attacker {
	a := &Attacker{
		stopch:  make(chan struct{}),
		workers: DefaultWorkers,
		maxBody: DefaultMaxBody,
		began:   time.Now(),

		metricFactory: metricFactory,
	}

	for _, opt := range opts {
		opt(a)
	}

	return a
}

// Workers returns a functional option which sets the initial number of workers
// an Attacker uses to hit its targets. More workers may be spawned dynamically
// to sustain the requested rate in the face of slow responses and errors.
func Workers(n uint64) func(*Attacker) {
	return func(a *Attacker) { a.workers = n }
}

// Timeout returns a functional option which sets the maximum amount of time
// an Attacker will wait for a request to be responded to and completely read.
func Timeout(d time.Duration) func(*Attacker) {
	return func(a *Attacker) {
		a.timeout = d
	}
}

// Attack reads its Targets from the passed Targeter and attacks them.
func (a *Attacker) Attack(tr Targeter, rate int, du time.Duration, name string) (*Metrics, error) {
	wg := sync.WaitGroup{}

	results, err := a.metricFactory()
	if err != nil {
		return nil, err
	}

	workerDelay := time.Duration(0)
	if rate != 0 {
		workerDelay = time.Duration(1e9 / rate * int(a.workers))
	}

	// Start workers.
	resultMutex := &sync.Mutex{}
	wg.Add(int(a.workers))
	for i := uint64(0); i < a.workers; i++ {
		go func(n int) {
			defer wg.Done()

			// Initial wait, so that not all workers start at once.
			initialDelayChan := time.After(workerDelay / time.Duration(a.workers) * time.Duration(n))
			select {
			case <-a.stopch:
			case <-initialDelayChan:
			}

			workerResults, workerErr := a.attack(tr, name, workerDelay)

			// Merge worker result.
			resultMutex.Lock()
			defer resultMutex.Unlock()

			if workerErr != nil {
				err = workerErr

				// Treat any worker error as fatal.
				a.Stop()
				return
			}

			if workerErr := results.Merge(workerResults); err != nil {
				err = workerErr
			}
		}(int(i))
	}

	// Stop the attack after requested duration.
	wg.Add(1)
	go func() {
		defer wg.Done()

		end := time.After(du)
		select {
		case <-a.stopch:
		case <-end:
			a.Stop()
		}
	}()

	wg.Wait()

	if err != nil {
		return nil, err
	}

	return results, nil
}

// Stop stops the current attack.
func (a *Attacker) Stop() {
	select {
	case <-a.stopch:
		return
	default:
		close(a.stopch)
	}
}

func (a *Attacker) attack(tr Targeter, name string, delay time.Duration) (*Metrics, error) {
	results, err := a.metricFactory()
	if err != nil {
		return nil, err
	}

	// Each worker has its own client, to
	// avoid critical section congestion.
	client := &fasthttp.Client{
		ReadTimeout:     DefaultTimeout,
		TLSConfig:       DefaultTLSConfig,
		MaxConnsPerHost: int(a.workers),
	}

	rng := rand.New(rand.NewSource(time.Now().Unix()))

	result := &Result{}
	hitAt := time.Now()

	for {
		select {
		case <-a.stopch:
			return results, nil
		default:
			a.hit(tr, name, client, result)
			if err := results.Add(result); err != nil {
				return nil, err
			}

			// Const rate pacer.
			if delay != 0 {
				randDelay := randomizeDelay(delay, rng)

				hitAt = hitAt.Add(randDelay)
				toWait := time.Until(hitAt)

				if toWait > 0 {
					delayChan := time.After(toWait)
					select {
					case <-a.stopch:
					case <-delayChan:
					}
				} else {
					// Hit is already overdue.
					hitAt = time.Now()
				}
			}
		}
	}
}

func (a *Attacker) hit(tr Targeter, name string, client *fasthttp.Client, res *Result) {
	tgtStack := Target{}
	err := error(nil)

	//nolint:gosec // making sure tgt is not moved to heap by the compiler to optimize allocation.
	tgt := (*Target)(gohacks.Noescape(unsafe.Pointer(&tgtStack)))

	res.Reset()
	res.Attack = name
	res.Timestamp = a.began.Add(time.Since(a.began))
	res.Seq = a.seq.Inc() - 1

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	defer func() {
		res.Latency = time.Since(res.Timestamp)
		if err != nil {
			res.Error = err.Error()
		}

		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	if err = tr(tgt); err != nil {
		a.Stop()
		return
	}

	req.Header.SetMethod(tgt.Method)
	req.SetRequestURI(tgt.URL)

	for k, vs := range tgt.Header {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}

	if host := tgt.Header.Get("Host"); host != "" {
		req.Header.SetHost(host)
	}

	res.Method = tgt.Method
	res.URL = tgt.URL

	if name != "" {
		req.Header.Set("X-Vegeta-Attack", name)
	}

	if a.chunked {
		req.SetBodyStream(bytes.NewReader(tgt.Body), -1)
	} else {
		req.SetBody(tgt.Body)
	}

	err = client.Do(req, resp)
	if err != nil {
		return
	}

	res.Body = resp.Body()
	if a.maxBody >= 0 {
		res.Body = res.Body[:a.maxBody]
	}

	res.BytesIn = uint64(len(res.Body))
	res.BytesOut = uint64(len(req.Body()))
	res.Code = uint16(resp.StatusCode())

	if res.Code < 200 || res.Code >= 400 {
		res.Error = http.StatusText(resp.StatusCode())
	}
}

func randomizeDelay(d time.Duration, rng *rand.Rand) time.Duration {
	const deviation = 0.2
	return time.Duration(float64(d) - deviation + 2*deviation*rng.Float64())
}
