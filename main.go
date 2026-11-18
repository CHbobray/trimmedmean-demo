// Command trimmedmean-demo shows how to use the trimmedmean package.
//
// By default it generates a sample of integers and a sample of floats,
// saves them as CSV files (so R can read exactly the same numbers), and
// prints the ordinary mean and the trimmed mean of each. With -data it
// reads your own numbers instead, and with -bootstrap it runs a bootstrap
// study of the trimmed mean as a robust estimator of central tendency.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/CHbobray/trimmedmean"
)

const unset = -1.0

type options struct {
	sampleSize  int
	seed        uint64
	trim        float64
	lowerTrim   float64
	upperTrim   float64
	dataFile    string
	outputDir   string
	bootstrap   bool
	bootSamples int
	iterations  int
	cpuProfile  string
	memProfile  string
}

func main() {
	opts := parseFlags()
	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		fmt.Fprintln(os.Stderr, "run with -h to see the available options")
		os.Exit(1)
	}
}

func parseFlags() options {
	var opts options
	flag.IntVar(&opts.sampleSize, "n", 100, "size of each generated sample (at least 100 for the assignment)")
	flag.Uint64Var(&opts.seed, "seed", 9999, "random seed, for reproducible results")
	flag.Float64Var(&opts.trim, "trim", 0.05, "symmetric trimming proportion, in [0, 0.5)")
	flag.Float64Var(&opts.lowerTrim, "lower", unset, "lower-tail trimming proportion (use with -upper for asymmetric trimming)")
	flag.Float64Var(&opts.upperTrim, "upper", unset, "upper-tail trimming proportion (use with -lower for asymmetric trimming)")
	flag.StringVar(&opts.dataFile, "data", "", "file of numbers, one per line, to use instead of generated samples")
	flag.StringVar(&opts.outputDir, "out", "data", "directory where generated samples are saved as CSV")
	flag.BoolVar(&opts.bootstrap, "bootstrap", false, "run the bootstrap study instead of the R comparison")
	flag.IntVar(&opts.bootSamples, "B", 100, "bootstrap samples per study sample")
	flag.IntVar(&opts.iterations, "iterations", 100, "independent samples per sample size in the bootstrap study")
	flag.StringVar(&opts.cpuProfile, "cpuprofile", "", "write a CPU profile to this file")
	flag.StringVar(&opts.memProfile, "memprofile", "", "write a heap profile to this file")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\nOptions:\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
	flag.Parse()
	return opts
}

func run(opts options) error {
	trims, err := trimArguments(opts)
	if err != nil {
		return err
	}
	if opts.cpuProfile != "" {
		stop, err := startCPUProfile(opts.cpuProfile)
		if err != nil {
			return err
		}
		defer stop()
	}

	start := time.Now()
	switch {
	case opts.bootstrap:
		err = runStudy(opts)
	case opts.dataFile != "":
		err = summarizeFile(opts.dataFile, trims)
	default:
		err = summarizeGenerated(opts, trims)
	}
	if err != nil {
		return err
	}

	logResourceUse(start)
	if opts.memProfile != "" {
		return writeHeapProfile(opts.memProfile)
	}
	return nil
}

// trimArguments validates the trimming flags and returns the arguments to
// pass to TrimmedMean: one proportion (symmetric) or two (lower, upper).
func trimArguments(opts options) ([]float64, error) {
	lowerSet := opts.lowerTrim != unset
	upperSet := opts.upperTrim != unset
	if lowerSet != upperSet {
		return nil, errors.New("-lower and -upper must be given together")
	}
	trims := []float64{opts.trim}
	if lowerSet {
		trims = []float64{opts.lowerTrim, opts.upperTrim}
	}
	// Check the proportions now so a bad value is reported before any work starts.
	if _, err := trimmedmean.TrimmedMean([]float64{0}, trims...); err != nil {
		return nil, err
	}
	if opts.bootstrap && lowerSet {
		return nil, errors.New("the bootstrap study uses symmetric trimming; use -trim instead of -lower/-upper")
	}
	return trims, nil
}

// summarizeGenerated creates the integer and float samples, saves them for
// R, and prints the Go results.
func summarizeGenerated(opts options, trims []float64) error {
	if opts.sampleSize < 1 {
		return fmt.Errorf("-n must be positive, got %d", opts.sampleSize)
	}
	rng := rand.New(rand.NewPCG(opts.seed, opts.seed))
	ints := generateInts(rng, opts.sampleSize)
	floats := generateFloats(rng, opts.sampleSize, defaultContamination)

	if err := os.MkdirAll(opts.outputDir, 0o755); err != nil {
		return err
	}
	intsPath := filepath.Join(opts.outputDir, "ints.csv")
	floatsPath := filepath.Join(opts.outputDir, "floats.csv")
	if err := writeNumbers(intsPath, ints); err != nil {
		return err
	}
	if err := writeNumbers(floatsPath, floats); err != nil {
		return err
	}

	fmt.Printf("Go trimmed mean results (seed %d, trim %v)\n", opts.seed, trims)
	if err := printSummary("integers "+intsPath, ints, trims); err != nil {
		return err
	}
	return printSummary("floats   "+floatsPath, floats, trims)
}

// summarizeFile prints results for numbers read from the user's file.
func summarizeFile(path string, trims []float64) error {
	values, err := readNumbers(path)
	if err != nil {
		return err
	}
	fmt.Printf("Go trimmed mean results (trim %v)\n", trims)
	return printSummary(path, values, trims)
}

func printSummary[T int | float64](label string, values []T, trims []float64) error {
	plain, err := trimmedmean.TrimmedMean(values)
	if err != nil {
		return err
	}
	trimmed, err := trimmedmean.TrimmedMean(values, trims...)
	if err != nil {
		return err
	}
	fmt.Printf("  %s  n = %d  mean = %.10f  trimmed mean = %.10f\n", label, len(values), plain, trimmed)
	return nil
}

func runStudy(opts options) error {
	if opts.bootSamples < 2 || opts.iterations < 2 {
		return errors.New("-B and -iterations must each be at least 2")
	}
	cfg := studyConfig{
		bootstrapSamples: opts.bootSamples,
		iterations:       opts.iterations,
		sampleSizes:      []int{25, 100, 225, 400},
		trim:             opts.trim,
		seed:             opts.seed,
		population:       defaultContamination,
	}
	return runBootstrapStudy(cfg, os.Stdout)
}

func startCPUProfile(path string) (stop func(), err error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		file.Close()
		return nil, err
	}
	return func() { pprof.StopCPUProfile(); file.Close() }, nil
}

func writeHeapProfile(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	runtime.GC()
	return pprof.WriteHeapProfile(file)
}

// logResourceUse reports elapsed time and memory to stderr, keeping
// stdout clean for results.
func logResourceUse(start time.Time) {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	log.Printf("elapsed %v, total allocated %.1f MiB, peak heap %.1f MiB",
		time.Since(start).Round(time.Millisecond),
		float64(mem.TotalAlloc)/(1<<20), float64(mem.HeapSys)/(1<<20))
}
