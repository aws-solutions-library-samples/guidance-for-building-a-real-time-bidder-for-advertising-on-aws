package main

import (
	"bytes"
	"log"
	"os"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

func main() {
	initConfig()
	config := loadConfig()

	var err error
	var inputs []vegeta.Metrics

	if len(config.InputFiles) > 0 {
		inputs, err = loadFromLocalFiles(config)
	} else {
		inputs, err = loadFromK8s(config)
	}

	if err != nil {
		log.Fatal("Unable to load inputs data: ", err)
	}

	aggregatedMetrics, err := aggregateMetrics(inputs)

	if err != nil {
		log.Fatal("Unable to aggregate data: ", err)
	}

	report := new(bytes.Buffer)

	mustWrite(report.WriteString("Aggregated report\n"))
	mustWrite(report.WriteString("-----------------\n"))

	reporter := vegeta.NewTextReporter(aggregatedMetrics)

	if err = reporter.Report(report); err != nil {
		log.Fatal(err)
	}

	mustWrite(report.WriteString("\n"))
	mustWrite(report.WriteString("Source reports\n"))
	mustWrite(report.WriteString("--------------\n\n"))

	for i := range inputs {
		reporter := vegeta.NewTextReporter(&inputs[i])

		if err = reporter.Report(report); err != nil {
			log.Fatal(err)
		}

		mustWrite(report.WriteString("\n"))
	}

	mustWrite(report.WriteTo(os.Stdout))
}

func mustWrite(_ interface{}, err error) {
	if err != nil {
		log.Fatal(err)
	}
}
