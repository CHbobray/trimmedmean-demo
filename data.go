package main

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"strconv"
	"strings"
)

// contaminationSpec describes a mixture of two normal distributions:
// most values come from the main population, a few from an outlier population.
// The defaults mirror run-bootstrap-robust.R.
type contaminationSpec struct {
	mainMean, mainSD       float64
	outlierMean, outlierSD float64
	outlierProbability     float64
}

var defaultContamination = contaminationSpec{
	mainMean: 100, mainSD: 10,
	outlierMean: 200, outlierSD: 1,
	outlierProbability: 0.01,
}

// draw returns one value from the mixture distribution.
func (c contaminationSpec) draw(rng *rand.Rand) float64 {
	if rng.Float64() >= c.outlierProbability {
		return c.mainMean + c.mainSD*rng.NormFloat64()
	}
	return c.outlierMean + c.outlierSD*rng.NormFloat64()
}

// generateFloats returns n values from the contaminated normal mixture.
func generateFloats(rng *rand.Rand, n int, spec contaminationSpec) []float64 {
	values := make([]float64, n)
	for i := range values {
		values[i] = spec.draw(rng)
	}
	return values
}

// generateInts returns n integer scores between 1 and 100, plus a handful
// of large outliers (about 3%) so that trimming has something to do.
func generateInts(rng *rand.Rand, n int) []int {
	values := make([]int, n)
	for i := range values {
		if rng.Float64() < 0.03 {
			values[i] = 1000 + rng.IntN(9000)
		} else {
			values[i] = 1 + rng.IntN(100)
		}
	}
	return values
}

// readNumbers reads one number per line from a text or single-column CSV
// file. Blank lines and lines starting with '#' are skipped; a header line
// that is not a number is allowed only as the first line.
func readNumbers(path string) ([]float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var values []float64
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		value, parseErr := strconv.ParseFloat(line, 64)
		if parseErr != nil {
			if lineNumber == 1 {
				continue // header row such as "x"
			}
			return nil, fmt.Errorf("%s line %d: %q is not a number", path, lineNumber, line)
		}
		values = append(values, value)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("%s contains no numbers", path)
	}
	return values, nil
}

// writeNumbers writes values one per line under a header "x", using the
// shortest representation that round-trips exactly, so R reads the same
// numbers Go used.
func writeNumbers[T int | float64](path string, values []T) error {
	var builder strings.Builder
	builder.WriteString("x\n")
	for _, v := range values {
		switch v := any(v).(type) {
		case int:
			builder.WriteString(strconv.Itoa(v))
		case float64:
			builder.WriteString(strconv.FormatFloat(v, 'g', -1, 64))
		}
		builder.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(builder.String()), 0o644)
}
