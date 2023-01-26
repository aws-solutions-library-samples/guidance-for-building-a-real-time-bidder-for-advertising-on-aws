package app

import (
	"context"
	"fmt"
	profilerServer "load-generator/code/diagnostic_server"
	loadgenerator "load-generator/code/load_generator"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/kelseyhightower/envconfig"
)

const histogramBucketRangeArguments = 3

// App used to generate load.
func App() error {
	rand.Seed(time.Now().Unix())
	initConfig()

	var conf = loadConfig()

	metricFactory := func() (*loadgenerator.Metrics, error) {
		histogram := loadgenerator.Histogram{}
		if err := createHistogramBuckets(&histogram, conf); err != nil {
			return nil, err
		}

		metric, err := loadgenerator.NewMetrics()
		if err != nil {
			return nil, err
		}

		metric.Histogram = &histogram
		return metric, nil
	}

	var loadTestName = fmt.Sprintf("Load test at %s", time.Now().UTC().String())

	attacker := loadgenerator.NewAttacker(
		metricFactory,
		loadgenerator.Workers(conf.Workers),
		loadgenerator.Timeout(conf.Timeout),
	)

	var finishProfiling = make(chan struct{})
	if conf.ProfilerEnabled {
		go func() {
			if err := collectProfile(conf, "profile", finishProfiling); err != nil {
				fmt.Println(err)
			}
		}()
		go func() {
			if err := collectProfile(conf, "allocs", finishProfiling); err != nil {
				fmt.Println(err)
			}
		}()
		go func() {
			if err := collectProfile(conf, "heap", finishProfiling); err != nil {
				fmt.Println(err)
			}
		}()
	}

	cfgProfServer := profilerServer.Config{}
	if err := envconfig.Process("", &cfgProfServer); err != nil {
		return errors.Wrap(err, "error during config initialization")
	}
	profServer := profilerServer.NewServer(cfgProfServer)

	profServer.AsyncListenAndServe(func(err error) {
		fmt.Println(errors.Wrap(err, "error during profiler server operation initialization"))
	})

	time.Sleep(conf.StartDelay)

	pool := &pool{}
	results, err := attacker.Attack(
		bidRequestTargeter(conf.Targets, conf.DevicesUsed, conf.NobidFraction, conf.OpenRTB3Fraction, pool),
		conf.Rate,
		conf.Duration,
		loadTestName,
	)
	if err != nil {
		return errors.Wrap(err, "error during load generation")
	}

	results.Close()

	if conf.ProfilerEnabled {
		finishProfiling <- struct{}{}
		finishProfiling <- struct{}{}
		finishProfiling <- struct{}{}
	}

	reporter := loadgenerator.NewJSONReporter(results)

	if err := reporter.Report(os.Stdout); err != nil {
		return errors.Wrap(err, "error during report generation")
	}

	if err := profServer.Shutdown(cfgProfServer.ShutdownTimeout); err != nil {
		return errors.Wrap(err, "error during profiler server shutdown")
	}

	return nil
}

func bidRequestTargeter(targets []string, devicesUsed int, nobidFraction, openrtb3Fraction float64, pool *pool) loadgenerator.Targeter {
	var index uint32 = 0

	return func(t *loadgenerator.Target) error {
		builder, err := pool.Get()
		if err != nil {
			return err
		}
		defer pool.Put(builder)

		body, version, err := builder.Generate(devicesUsed, nobidFraction, openrtb3Fraction)
		if err != nil {
			return errors.Wrap(err, "error during request generation")
		}

		targetIndex := atomic.AddUint32(&index, 1) % uint32(len(targets))

		t.Method = "POST"
		t.URL = targets[targetIndex]
		t.Body = body
		t.Header = http.Header{}
		t.Header.Add("x-openrtb-version", version)

		return nil
	}
}

// collectProfile requests profiler data and saves it to the output file
func collectProfile(conf *config, endpoint string, finish <-chan struct{}) error {
	defer func() {
		<-finish
	}()
	profilerURL := *conf.ProfilerURL
	profilerURL.Path += endpoint

	profilerKey := strings.Builder{}
	hostname, err := os.Hostname()
	if err != nil {
		log.Println("cannot obtain hostname for profiler key, leaving empty:", err)
	}
	err = conf.ProfilerOutput.Execute(&profilerKey, keyData{Endpoint: endpoint, Hostname: hostname, Config: conf})
	if err != nil {
		return errors.Wrap(err, "invalid profiler output template")
	}

	resp, err := http.Get(profilerURL.String())
	if err != nil {
		return errors.Wrap(err, "invalid profiler URL")
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.Println(errors.Wrap(err, "error closing response body"))
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected profiler status: %s", resp.Status)
	}

	cfg, err := awsConfig.LoadDefaultConfig(context.Background())
	if err != nil {
		return errors.Wrap(err, "cannot initialize AWS config")
	}
	client := s3.NewFromConfig(cfg)
	uploader := manager.NewUploader(client)

	_, err = uploader.Upload(context.Background(), &s3.PutObjectInput{
		Body:   resp.Body,
		Bucket: &conf.ProfilerBucket,
		Key:    aws.String(profilerKey.String()),
	})
	if err != nil {
		return errors.Wrap(err, "cannot write results to S3")
	}

	return nil
}

// createHistogramBuckets create histogram buckets from specification
func createHistogramBuckets(hist *loadgenerator.Histogram, conf *config) error {
	bucketRanges := strings.Split(conf.HistogramSpec, ",")

	for _, bucketRange := range bucketRanges {
		rawParams := strings.Split(bucketRange, ":")

		if len(rawParams) != histogramBucketRangeArguments {
			return errors.Errorf("Invalid histogram specification. Invalid range: %s", bucketRange)
		}

		params := make([]time.Duration, 0)

		for _, rawParam := range rawParams {
			val, err := time.ParseDuration(rawParam)

			if err != nil {
				return errors.Errorf("Invalid histogram specification. Invalid duration provided: %s", rawParam)
			}

			params = append(params, val)
		}

		for i := params[0]; i <= params[1]; i += params[2] {
			hist.Buckets = append(hist.Buckets, i)
		}
	}

	return nil
}
