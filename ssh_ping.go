// ssh_ping is a utility for measuring roundtrip latency of SSH connections.
//
// Run it as follows:
//
//	ssh_ping --host foo.bar.com
//
// This will make an SSH connection, then repeatedly send data to be echoed
// back to the client, measuring statistics about how long echoing takes. Stats
// are collected for five seconds and then printed to stdout.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/montanaflynn/stats"
)

var host = flag.String("host", "", "Host to connect to over SSH.")

func formatMillis(d time.Duration) string {
	return fmt.Sprintf("%4.1f ms", float64(d.Round(100*time.Microsecond))/float64(time.Millisecond))
}

func toFloatSeconds(s []time.Duration) []float64 {
	result := make([]float64, 0, len(s))
	for _, d := range s {
		result = append(result, float64(d)/float64(time.Second))
	}

	return result
}

func computeDurationStat(compute func(stats.Float64Data) (float64, error), s []time.Duration) time.Duration {
	seconds, err := compute(toFloatSeconds(s))
	if err != nil {
		log.Fatal(err)
	}

	return time.Duration(seconds * float64(time.Second))
}

func min(s []time.Duration) time.Duration {
	return computeDurationStat(stats.Min, s)
}

func median(s []time.Duration) time.Duration {
	return computeDurationStat(stats.Median, s)
}

func percentile(percent float64, s []time.Duration) time.Duration {
	return computeDurationStat(func(data stats.Float64Data) (float64, error) { return stats.Percentile(data, percent) }, s)
}

func max(s []time.Duration) time.Duration {
	return computeDurationStat(stats.Max, s)
}

func mean(s []time.Duration) time.Duration {
	return computeDurationStat(stats.Mean, s)
}

func stdDev(s []time.Duration) time.Duration {
	return computeDurationStat(stats.StandardDeviation, s)
}

func runPing(outgoing io.Writer, incoming io.Reader) (d time.Duration, err error) {
	start := time.Now()

	// Write a magic string.
	_, err = io.Copy(outgoing, bytes.NewBufferString("foo\n"))
	if err != nil {
		return
	}

	// Wait for it to be echoed back.
	buf := make([]byte, 4)
	_, err = io.ReadFull(incoming, buf)
	if err != nil {
		return
	}

	d = time.Since(start)
	return
}

func main() {
	flag.Parse()

	if *host == "" {
		fmt.Fprintf(os.Stderr, "Must set --host.\n")
		os.Exit(1)
	}

	// Start an ssh command that echoes whatever we write to it.
	cmd := exec.Command("ssh", *host, "--", "cat")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	defer stdin.Close()

	// The first few pings probably incur some startup cost. Throw them away.
	for i := 0; i < 3; i++ {
		if _, err := runPing(stdin, stdout); err != nil {
			log.Fatal(err)
		}
	}

	// Collect samples for 5 seconds.
	samples := []time.Duration{}
	for start := time.Now(); time.Since(start) < 5*time.Second; {
		sample, err := runPing(stdin, stdout)
		if err != nil {
			log.Fatal(err)
		}

		samples = append(samples, sample)
		if len(samples)%100 == 0 {
			fmt.Println(len(samples), "samples so far...")
		}
	}

	fmt.Printf("Collected %d samples.\n", len(samples))
	fmt.Printf("\n")
	fmt.Printf("Min:      %s\n", formatMillis(min(samples)))
	fmt.Printf("p05:      %s\n", formatMillis(percentile(5, samples)))
	fmt.Printf("p50:      %s\n", formatMillis(median(samples)))
	fmt.Printf("p95:      %s\n", formatMillis(percentile(95, samples)))
	fmt.Printf("Max:      %s\n", formatMillis(max(samples)))
	fmt.Printf("\n")
	fmt.Printf("Mean:     %s\n", formatMillis(mean(samples)))
	fmt.Printf("Std. dev: %s\n", formatMillis(stdDev(samples)))
}
