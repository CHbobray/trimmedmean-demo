package main

import (
	"os"
	"path/filepath"
	"testing"
)

// defaultOptions mirrors the command-line defaults, writing into a temporary directory.
func defaultOptions(t *testing.T) options {
	return options{
		sampleSize: 100, seed: 9999, trim: 0.05,
		lowerTrim: unset, upperTrim: unset,
		outputDir: t.TempDir(), bootSamples: 10, iterations: 3,
	}
}

func TestRunGeneratesSamplesForR(t *testing.T) {
	opts := defaultOptions(t)
	if err := run(opts); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ints.csv", "floats.csv"} {
		values, err := readNumbers(filepath.Join(opts.outputDir, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(values) != opts.sampleSize {
			t.Errorf("%s: %d values, want %d", name, len(values), opts.sampleSize)
		}
	}
}

func TestRunWithDataFileAndAsymmetricTrim(t *testing.T) {
	opts := defaultOptions(t)
	opts.dataFile = filepath.Join(opts.outputDir, "mine.csv")
	opts.lowerTrim, opts.upperTrim = 0.1, 0
	if err := os.WriteFile(opts.dataFile, []byte("x\n1\n2\n3\n4\n5\n6\n7\n8\n9\n100\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(opts); err != nil {
		t.Fatal(err)
	}
}

func TestRunBootstrapWithProfiles(t *testing.T) {
	opts := defaultOptions(t)
	opts.bootstrap = true
	opts.cpuProfile = filepath.Join(opts.outputDir, "cpu.prof")
	opts.memProfile = filepath.Join(opts.outputDir, "mem.prof")
	if err := run(opts); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{opts.cpuProfile, opts.memProfile} {
		if info, err := os.Stat(path); err != nil || info.Size() == 0 {
			t.Errorf("profile %s was not written", path)
		}
	}
}

func TestRunRejectsInvalidInput(t *testing.T) {
	cases := map[string]func(*options){
		"zero sample size":   func(o *options) { o.sampleSize = 0 },
		"trim out of range":  func(o *options) { o.trim = 0.5 },
		"missing data file":  func(o *options) { o.dataFile = "does-not-exist.csv" },
		"too few resamples":  func(o *options) { o.bootstrap, o.bootSamples = true, 1 },
		"too few iterations": func(o *options) { o.bootstrap, o.iterations = true, 1 },
		"unwritable profile": func(o *options) { o.cpuProfile = filepath.Join(o.outputDir, "no", "such", "dir", "cpu.prof") },
	}
	for name, change := range cases {
		opts := defaultOptions(t)
		change(&opts)
		if err := run(opts); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
