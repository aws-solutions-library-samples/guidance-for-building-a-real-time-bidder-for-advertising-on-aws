package app

import (
	"fmt"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type config struct {
	Rate             int
	Slope            float64
	Duration         time.Duration
	Timeout          time.Duration
	StartDelay       time.Duration
	Targets          []string
	DevicesUsed      int
	NobidFraction    float64
	Workers          uint64
	MaxWorkers       uint64
	ProfilerURL      *url.URL
	ProfilerBucket   string
	ProfilerOutput   *template.Template
	ProfilerEnabled  bool
	HistogramSpec    string
	TrackErrors      bool
	OpenRTB3Fraction float64
}

// keyData contains fields accessible in profiler output templates.
type keyData struct {
	// Endpoint is the profiler endpoint e.g. profile, allocs, heap.
	Endpoint string
	// Hostname is the host name of `os.Hostname`; empty on error.
	Hostname string
	// Config is the load generator config.
	Config *config
}

const defaultDurationSeconds = 6
const defaultTimeoutMilliseconds = 100
const defaultStartDelaySeconds = 1
const defaultHistogramSpec = "0ms:29ms:1ms,30ms:100ms:2ms,110ms:1000ms:10ms"

func initConfig() {
	pflag.Int("initial-rate", 100, "initial number of requests per second (increases linearly if a positive slope is provided)")
	pflag.Float64("slope", 0.0, "slope of requests per second increase (zero for a constant rate; see <https://en.wikipedia.org/wiki/Slope>)")
	pflag.Duration("duration", defaultDurationSeconds*time.Second, "Duration of load test")
	pflag.Duration("timeout", defaultTimeoutMilliseconds*time.Millisecond, "Request timeout")
	pflag.Duration("start-delay", defaultStartDelaySeconds*time.Second, "Time to wait before starting the benchmark")
	pflag.StringSlice("target", []string{""}, "URL to the bidder, usually ending with /bidrequest")
	pflag.Int("devices-used", 10, "number of unique device ids that can be generated during load tests")
	pflag.Float64("nobid-fraction", 0.1,
		"fraction of bidrequests that should provoke nobid response from bidder by using device ID not stored int the database")
	pflag.Uint("workers", 10, "the initial number of workers used in the attack")
	pflag.Uint("max-workers", 18446744073709551615, "the maximum number of workers used in the attack")
	pflag.String("profiler-url", "", "base URL to the profiler, usually ending with /debug/pprof/")
	pflag.String("profiler-bucket", "", "S3 bucket to save pprof output to, pass a non-empty value to enable profiling")
	pflag.String("profiler-output", "", "template of S3 key to save pprof output to, pass a non-empty value to enable profiling")
	pflag.String("histogram", defaultHistogramSpec, "histogram buckets specification in form of 'FROM:TO:SIZE,FROM:TO:SIZE...'")
	pflag.Bool("track-errors", false, "enable request error tracking (can produce a lot of output)")
	pflag.Float64("openrtb3-fraction", 1.0,
		"fraction of bid requests in OpenRTB 3.0, others use OpenRTB 2.5; set to 1.0 to use only 3.0, to 0.0 to use only 2.5")
	pflag.Parse()

	if err := viper.BindPFlags(pflag.CommandLine); err != nil {
		panic(err)
	}

	viper.SetEnvPrefix("load_generator")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func loadConfig() *config {
	targets := viper.GetStringSlice("target")
	duration := viper.GetDuration("duration")
	startDelay := viper.GetDuration("start-delay")
	rawProfilerURL := viper.GetString("profiler-url")
	profilerURL := (*url.URL)(nil)

	_, err := url.Parse(targets[0])

	if err != nil {
		panic(err)
	}

	if rawProfilerURL != "" {
		profilerURL, err = url.Parse(rawProfilerURL)

		if err != nil {
			panic(err)
		}

		durationWithDelay := duration + startDelay
		profilerURL.RawQuery = fmt.Sprintf("seconds=%v", durationWithDelay.Seconds())
	}

	profilerBucket := viper.GetString("profiler-bucket")
	profilerOutputText := viper.GetString("profiler-output")
	var profilerOutput *template.Template
	if profilerOutputText != "" {
		profilerOutput, err = template.New("profiler-output").Parse(profilerOutputText)
		if err != nil {
			panic(err)
		}
	}

	return &config{
		Rate:             viper.GetInt("initial-rate"),
		Slope:            viper.GetFloat64("slope"),
		Duration:         duration,
		Timeout:          viper.GetDuration("timeout"),
		StartDelay:       startDelay,
		Targets:          targets,
		DevicesUsed:      viper.GetInt("devices-used"),
		NobidFraction:    viper.GetFloat64("nobid-fraction"),
		Workers:          viper.GetUint64("workers"),
		MaxWorkers:       viper.GetUint64("max-workers"),
		ProfilerURL:      profilerURL,
		ProfilerBucket:   profilerBucket,
		ProfilerOutput:   profilerOutput,
		ProfilerEnabled:  profilerURL != nil && profilerBucket != "" && profilerOutput != nil,
		HistogramSpec:    viper.GetString("histogram"),
		TrackErrors:      viper.GetBool("track-errors"),
		OpenRTB3Fraction: viper.GetFloat64("openrtb3-fraction"),
	}
}
