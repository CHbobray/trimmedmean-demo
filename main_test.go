package main

import (
	"bytes"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestMedianMatchesR(t *testing.T) {
	cases := []struct {
		values []float64
		want   float64
	}{
		{[]float64{3, 1, 2}, 2},
		{[]float64{4, 1, 3, 2}, 2.5},
		{[]float64{7}, 7},
		{[]float64{5, 5, 1, 100}, 5},
	}
	for _, tc := range cases {
		input := slices.Clone(tc.values)
		if got := median(input); got != tc.want {
			t.Errorf("median(%v) = %v, want %v", tc.values, got, tc.want)
		}
		if !slices.Equal(input, tc.values) {
			t.Errorf("median modified its input")
		}
	}
}

func TestStandardDeviationMatchesR(t *testing.T) {
	// R: sd(c(2, 4, 4, 4, 5, 5, 7, 9)) = 2.138089935299395
	got := standardDeviation([]float64{2, 4, 4, 4, 5, 5, 7, 9})
	if math.Abs(got-2.138089935299395) > 1e-12 {
		t.Errorf("got %v", got)
	}
	if !math.IsNaN(standardDeviation([]float64{1})) {
		t.Errorf("sd of one value should be NaN, as in R")
	}
}

func TestWriteThenReadRoundTrips(t *testing.T) {
	dir := t.TempDir()
	floats := generateFloats(rand.New(rand.NewPCG(1, 1)), 150, defaultContamination)
	path := filepath.Join(dir, "floats.csv")
	if err := writeNumbers(path, floats); err != nil {
		t.Fatal(err)
	}
	got, err := readNumbers(path)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, floats) {
		t.Errorf("values changed in the round trip")
	}
}

func TestReadNumbersRejectsBadInput(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"bad value": "x\n1\n2\nthree\n",
		"empty":     "x\n\n# only a comment\n",
	}
	for name, content := range cases {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "_"))
		os.WriteFile(path, []byte(content), 0o644)
		if _, err := readNumbers(path); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
	if _, err := readNumbers(filepath.Join(dir, "missing.csv")); err == nil {
		t.Errorf("missing file: expected an error")
	}
}

func TestTrimArguments(t *testing.T) {
	base := options{trim: 0.05, lowerTrim: unset, upperTrim: unset}

	if got, err := trimArguments(base); err != nil || !slices.Equal(got, []float64{0.05}) {
		t.Errorf("symmetric: got %v, %v", got, err)
	}

	asym := base
	asym.lowerTrim, asym.upperTrim = 0.1, 0.0
	if got, err := trimArguments(asym); err != nil || !slices.Equal(got, []float64{0.1, 0}) {
		t.Errorf("asymmetric: got %v, %v", got, err)
	}

	invalid := []options{
		{trim: 0.6, lowerTrim: unset, upperTrim: unset},               // out of range
		{trim: 0.05, lowerTrim: 0.1, upperTrim: unset},                // only one side given
		{trim: 0.05, lowerTrim: 0.1, upperTrim: 0.1, bootstrap: true}, // study is symmetric
	}
	for _, opts := range invalid {
		if _, err := trimArguments(opts); err == nil {
			t.Errorf("%+v: expected an error", opts)
		}
	}
}

func TestBootstrapStudyRunsAndIsReproducible(t *testing.T) {
	cfg := studyConfig{
		bootstrapSamples: 20, iterations: 5, sampleSizes: []int{25, 100},
		trim: 0.05, seed: 42, population: defaultContamination,
	}
	var first, second bytes.Buffer
	if err := runBootstrapStudy(cfg, &first); err != nil {
		t.Fatal(err)
	}
	runBootstrapStudy(cfg, &second)
	if first.String() != second.String() {
		t.Errorf("same seed gave different output")
	}
	if !strings.Contains(first.String(), "Samples of size n = 100") {
		t.Errorf("unexpected output:\n%s", first.String())
	}
}

func BenchmarkBootstrapStudy(b *testing.B) {
	cfg := studyConfig{
		bootstrapSamples: 100, iterations: 10, sampleSizes: []int{25, 100, 225, 400},
		trim: 0.05, seed: 9999, population: defaultContamination,
	}
	b.ReportAllocs()
	for b.Loop() {
		runBootstrapStudy(cfg, &bytes.Buffer{})
	}
}
