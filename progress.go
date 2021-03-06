package main

import (
	"io"

	"gopkg.in/cheggaaa/pb.v1"
)

type progressReporter struct {
	verbose bool
	bar     *pb.ProgressBar
}

func makeProgressReporter(verbose bool) *progressReporter {
	result := new(progressReporter)
	result.verbose = verbose
	return result
}

func (pr progressReporter) IsProgressReportingRequested() bool {
	return pr.verbose
}

func (pr *progressReporter) StartReportableActivity(expectedItems int) {
	pr.bar = pb.StartNew(expectedItems)
	pr.bar.ShowCounters = true
}

func (pr *progressReporter) StartReportableReaderActivityInBytes(exepectedBytes int64, inputReader io.Reader) io.Reader {
	pr.bar = pb.New(int(exepectedBytes)).SetUnits(pb.U_BYTES)
	pr.bar.Start()
	return pr.bar.NewProxyReader(inputReader)
}

func (pr *progressReporter) IncrementReportableActivityProgress() {
	pr.bar.Increment()
}

func (pr *progressReporter) IncrementReportableActivityProgressBy(incrementBy int) {
	pr.bar.Add(incrementBy)
}

func (pr *progressReporter) CompleteReportableActivityProgress(summary string) {
	pr.bar.FinishPrint(summary)
}
