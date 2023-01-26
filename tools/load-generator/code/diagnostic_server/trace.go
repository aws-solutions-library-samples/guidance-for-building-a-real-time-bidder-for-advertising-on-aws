package diagnosticserver

import (
	"bytes"
	"log"
	"net/http"
	"runtime/trace"
	"time"
)

const defaultDuration = "5s"

// traceHandler collects program execution trace for
// requested duration and sends it in response body.
// Example query string with duration set:
// http: //localhost:8091/debug/trace?duration=1s
func traceHandler(resp http.ResponseWriter, req *http.Request) {
	log.Printf("received trace request")

	durationStr := defaultDuration

	durationQuery := req.URL.Query()["duration"]
	if len(durationQuery) > 0 && durationQuery[0] != "" {
		durationStr = durationQuery[0]
	}

	responseBody, err := runTrace(durationStr)
	if err != nil {
		log.Println("error while serving trace", err)
		responseBody = []byte(err.Error())
		resp.WriteHeader(http.StatusInternalServerError)
	}

	_, err = resp.Write(responseBody)
	if err != nil {
		log.Println("error while sending trace response", err)
	}
}

func runTrace(durationStr string) ([]byte, error) {
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		return nil, err
	}

	traceFile := bytes.Buffer{}
	err = trace.Start(&traceFile)
	if err != nil {
		return nil, err
	}

	time.Sleep(duration)

	trace.Stop()

	return traceFile.Bytes(), nil
}
