package main

import (
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/CHbobray/trimmedmean"
)

// studyConfig holds the settings for the bootstrap study, a Go port of
// run-bootstrap-robust.R by Thomas W. Miller.
type studyConfig struct {
	bootstrapSamples int     // B: resamples drawn from each sample
	iterations       int     // independent samples drawn per sample size
	sampleSizes      []int   // n values studied
	trim             float64 // symmetric trimming proportion
	seed             uint64
	population       contaminationSpec
}

// estimatorSummary collects, for one sample size, the statistic of interest
// from every iteration so that their spread can be measured.
type estimatorSummary struct {
	sampleMeans      []float64 // mean of each original sample
	bootMeans        []float64 // average of the B bootstrap means
	bootMedians      []float64 // average of the B bootstrap medians
	bootTrimmedMeans []float64 // average of the B bootstrap trimmed means
	withinSEMean     []float64 // SD of the B bootstrap means (classic bootstrap SE)
	withinSEMedian   []float64
	withinSETrimmed  []float64
}

// runBootstrapStudy repeats the R study: for each iteration and sample size
// it draws a contaminated sample, resamples it B times, and records the mean,
// median and trimmed mean. The standard error of each estimator is then the
// standard deviation of its values across iterations.
func runBootstrapStudy(cfg studyConfig, out io.Writer) error {
	rng := rand.New(rand.NewPCG(cfg.seed, cfg.seed))
	summaries := make(map[int]*estimatorSummary, len(cfg.sampleSizes))
	for _, n := range cfg.sampleSizes {
		summaries[n] = &estimatorSummary{}
	}

	for range cfg.iterations {
		for _, n := range cfg.sampleSizes {
			sample := generateFloats(rng, n, cfg.population)
			if err := resampleOnce(rng, sample, cfg, summaries[n]); err != nil {
				return err
			}
		}
	}

	printStudy(out, cfg, summaries)
	return nil
}

// resampleOnce draws B bootstrap samples from sample and appends the
// averaged statistics and within-sample bootstrap SEs to summary.
func resampleOnce(rng *rand.Rand, sample []float64, cfg studyConfig, summary *estimatorSummary) error {
	n := len(sample)
	means := make([]float64, cfg.bootstrapSamples)
	medians := make([]float64, cfg.bootstrapSamples)
	trimmedMeans := make([]float64, cfg.bootstrapSamples)
	resample := make([]float64, n)

	for b := range cfg.bootstrapSamples {
		for i := range resample {
			resample[i] = sample[rng.IntN(n)] // sampling with replacement
		}
		var err error
		if means[b], err = trimmedmean.TrimmedMean(resample); err != nil {
			return err
		}
		if trimmedMeans[b], err = trimmedmean.TrimmedMean(resample, cfg.trim); err != nil {
			return err
		}
		medians[b] = median(resample)
	}

	sampleMean, _ := trimmedmean.TrimmedMean(sample)
	summary.sampleMeans = append(summary.sampleMeans, sampleMean)
	summary.bootMeans = append(summary.bootMeans, average(means))
	summary.bootMedians = append(summary.bootMedians, average(medians))
	summary.bootTrimmedMeans = append(summary.bootTrimmedMeans, average(trimmedMeans))
	summary.withinSEMean = append(summary.withinSEMean, standardDeviation(means))
	summary.withinSEMedian = append(summary.withinSEMedian, standardDeviation(medians))
	summary.withinSETrimmed = append(summary.withinSETrimmed, standardDeviation(trimmedMeans))
	return nil
}

func printStudy(out io.Writer, cfg studyConfig, summaries map[int]*estimatorSummary) {
	p := cfg.population
	fmt.Fprintf(out, "\nStudy conditions:\n")
	fmt.Fprintf(out, "  Main population mean: %g SD: %g\n", p.mainMean, p.mainSD)
	fmt.Fprintf(out, "  Outlier mean: %g SD: %g\n", p.outlierMean, p.outlierSD)
	fmt.Fprintf(out, "  Probability of outliers: %g\n", p.outlierProbability)
	fmt.Fprintf(out, "\nTrimmed mean computed with trimming proportion %g\n", cfg.trim)
	fmt.Fprintf(out, "\nEstimated standard errors using %d bootstrap samples (%d iterations, seed %d)\n",
		cfg.bootstrapSamples, cfg.iterations, cfg.seed)

	for _, n := range cfg.sampleSizes {
		s := summaries[n]
		fmt.Fprintf(out, "\nSamples of size n = %d\n", n)
		fmt.Fprintf(out, "  SE Mean from Central Limit Theorem (assuming no outliers): %.2f\n", p.mainSD/math.Sqrt(float64(n)))
		fmt.Fprintf(out, "  SE Mean from Samples (with outliers): %.2f\n", standardDeviation(s.sampleMeans))
		fmt.Fprintf(out, "  SE Mean from Bootstrap Samples (with outliers): %.2f\n", standardDeviation(s.bootMeans))
		fmt.Fprintf(out, "  SE Median from Bootstrap Samples (with outliers): %.2f\n", standardDeviation(s.bootMedians))
		fmt.Fprintf(out, "  SE Trimmed Mean from Bootstrap Samples (with outliers): %.2f\n", standardDeviation(s.bootTrimmedMeans))
		fmt.Fprintf(out, "  Classic within-sample bootstrap SE (average over iterations): mean %.2f, median %.2f, trimmed mean %.2f\n",
			average(s.withinSEMean), average(s.withinSEMedian), average(s.withinSETrimmed))
	}
}

// average returns the arithmetic mean of a non-empty slice.
func average(values []float64) float64 {
	m, _ := trimmedmean.TrimmedMean(values)
	return m
}

// standardDeviation returns the sample standard deviation (n-1 denominator),
// matching R's sd().
func standardDeviation(values []float64) float64 {
	if len(values) < 2 {
		return math.NaN()
	}
	m := average(values)
	var sumSquares float64
	for _, v := range values {
		sumSquares += (v - m) * (v - m)
	}
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

// median returns the middle value of a non-empty slice, averaging the two
// middle values when the length is even (as R's median() does).
func median(values []float64) float64 {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 1 {
		return sorted[mid]
	}
	return (sorted[mid-1] + sorted[mid]) / 2
}
