package generator

import (
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sync"
)

// Device object has a `DeviceID` key and the concatenation of 16 byte audience ids
type Device struct {
	DeviceID    Key `dynamodbav:"d"`
	AudienceIds Key `dynamodbav:"a"`
}

// DeviceGenerator creates series of `Device` and writes to the `out` channel
func DeviceGenerator(out chan<- Record, cfg *Config, enc *Encryptor) {
	defer close(out)

	for i := cfg.KeyLow; i <= cfg.KeyHigh; i++ {
		out <- &Device{DeviceID: enc.Encrypt(i), AudienceIds: AudiencesConcat(i, cfg, enc)}
	}
}

// DevicePrinter receives the `Device` items from `in` channel,
// and writes to the `w` `io.Writer`
func DevicePrinter(in <-chan Record, w io.Writer) error {
	var err error
	var d *Device
	for r := range in {
		d = r.(*Device)
		if _, err = fmt.Fprintf(w, "%s", hex.EncodeToString(d.DeviceID)); err != nil {
			return err
		}
		if _, err = fmt.Fprintf(w, "\t%s", hex.EncodeToString(d.AudienceIds)); err != nil {
			return err
		}
		if _, err = fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

// GenerateDevices builds and runs the device generation pipeline
func GenerateDevices(cfg *Config) error {
	enc, err := NewDefaultEncryptor()
	if err != nil {
		return err
	}

	ch := make(chan Record)

	go DeviceGenerator(ch, cfg, enc)

	switch cfg.Output {
	case OutputStdout:
		return DevicePrinter(ch, os.Stdout)
	case OutputDynamodb, OutputAerospike:
		var last error
		worker := func(wg *sync.WaitGroup) {
			defer wg.Done()
			if err := writer(ch, cfg); err != nil {
				fmt.Printf("error: %v\n", err)
				last = err
			}
		}
		wg := workGroup(worker, cfg.DynamodbConcurrency)
		wg.Wait()
		return last
	default:
		return fmt.Errorf("unknown output: %v", cfg.Output)
	}
}
