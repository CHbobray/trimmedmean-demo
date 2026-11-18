# trimmedmean-demo

An example Go program that uses the
[`trimmedmean`](https://github.com/CHbobray/trimmedmean) package. It:

1. computes trimmed means of a sample of **100 integers** and a sample of **100 floats**,
2. saves both samples so R can read exactly the same numbers, and checks the Go results
   against R's `mean(x, trim = 0.05)`,
3. (optional part of the assignment) runs a **bootstrap study** of the trimmed mean as a
   robust estimator of central tendency, a Go port of the instructor's `run-bootstrap-robust.R`.

## Summary of results

**Go matches R exactly.** On the assignment samples (n = 100, symmetric trimming 0.05):

| Sample | Ordinary mean | Go trimmed mean | R `mean(x, trim = 0.05)` |
|---|---:|---:|---:|
| 100 integers (`data/ints.csv`) | 215.1000000000 | 59.1111111111 | 59.1111111111 |
| 100 floats (`data/floats.csv`) | 101.6331992242 | 99.7898634777 | 99.7898634777 |

The integers are scores from 1 to 100 with about 3% large outliers (1,000 to 9,999);
the floats come from the same contaminated normal mixture as the R study
(N(100, 10) with 1% chance of N(200, 1)). Trimming 5% from each end removes the outliers'
pull: the integer mean drops from 215 to 59, close to the center of the 1–100 scores.

A wider check (`crosscheck.sh`) compared **80 samples** (40 integer, 40 float) with sizes
from 137 to 1,580: every Go result equals R's to 10 decimal places
(`results/crosscheck.txt`). The package's own unit tests also use R-generated expected values.

**Bootstrap study.** Standard errors from 100 iterations with B = 100 bootstrap samples,
side by side with the R listing. Go and R use different random number generators, so the
numbers differ slightly, but the pattern is the same:

| n | Estimator | R listing | Go |
|---:|---|---:|---:|
| 25 | Mean (CLT, no outliers) | 2.00 | 2.00 |
| | Mean (with outliers) | 2.65 | 2.76 |
| | Median | 2.17 | 2.33 |
| | Trimmed mean (0.05) | 2.25 | 2.36 |
| 100 | Mean (CLT, no outliers) | 1.00 | 1.00 |
| | Mean (with outliers) | 1.42 | 1.49 |
| | Median | 1.11 | 1.28 |
| | Trimmed mean (0.05) | **1.02** | **1.15** |
| 225 | Mean (CLT, no outliers) | 0.67 | 0.67 |
| | Mean (with outliers) | 0.91 | 0.85 |
| | Median | 0.72 | 0.86 |
| | Trimmed mean (0.05) | **0.63** | **0.65** |
| 400 | Mean (CLT, no outliers) | 0.50 | 0.50 |
| | Mean (with outliers) | 0.75 | 0.66 |
| | Median | 0.68 | 0.63 |
| | Trimmed mean (0.05) | **0.57** | **0.52** |

Takeaways, which hold in both languages:

- Just 1% contamination inflates the standard error of the mean well above the
  Central Limit Theorem value (for example 0.66–0.75 instead of 0.50 at n = 400).
- Both robust estimators beat the mean. For n ≥ 100 the trimmed mean has the smallest
  standard error, close to what the mean would have with no outliers at all.
- The trimmed mean beats the median because it still averages 90% of the data, while the
  median uses only the middle value(s). This is the reason to prefer it as a robust
  estimator of central tendency when the contamination is light.

Full output: `results/bootstrap-B100.txt`, `results/bootstrap-B1000.txt`, and the R run
`results/r-bootstrap-B100.txt`. The Go output adds one extra line per sample size, the
classic bootstrap SE (the standard deviation of the B resampled statistics within one
sample, averaged over iterations), which estimates the same quantity from a single
sample, as you would in practice.

**Speed.** The study with B = 100 takes about 0.8 s in Go versus about 12 s for the R
script on the same machine; B = 1,000 takes about 8 s in Go.

## Getting and using the trimmedmean package in your own Go program

1. Create a module for your program (skip if you already have one):

   ```bash
   mkdir myprogram && cd myprogram
   go mod init example.com/myprogram
   ```

2. Get the package:

   ```bash
   go get github.com/CHbobray/trimmedmean@latest
   ```

3. Import and call it:

   ```go
   import "github.com/CHbobray/trimmedmean"

   m, err := trimmedmean.TrimmedMean(data, 0.05)        // symmetric, like R's mean(x, trim = 0.05)
   m, err = trimmedmean.TrimmedMean(data, 0.10, 0.02)   // asymmetric: 10% low end, 2% high end
   m, err = trimmedmean.TrimmedMean(data)               // ordinary mean
   ```

   `data` can be a slice of any integer or float type. Proportions must be in `[0, 0.5)`;
   invalid input returns an error rather than panicking. See the
   [package README](https://github.com/CHbobray/trimmedmean) for the full API.

4. Build and run: `go run .`

## Running this program

Requires Go 1.24 or later. R is needed only for the comparison scripts.

```bash
git clone https://github.com/CHbobray/trimmedmean-demo.git
cd trimmedmean-demo
go run .                          # trimmed means of 100 ints and 100 floats; writes data/*.csv
Rscript compare.R                 # R's mean(x, trim = 0.05) on the same files
./crosscheck.sh                   # Go vs R on 80 samples of varying size
go run . -bootstrap               # bootstrap study, B = 100
go run . -bootstrap -B 1000       # bootstrap study, B = 1000
```

### Options

| Flag | Default | Meaning |
|---|---|---|
| `-n` | 100 | size of each generated sample |
| `-seed` | 9999 | random seed (same seed, same results) |
| `-trim` | 0.05 | symmetric trimming proportion, in [0, 0.5) |
| `-lower`, `-upper` | — | asymmetric trimming; give both |
| `-data FILE` | — | use your own numbers (one per line; a header line is allowed) |
| `-out DIR` | `data` | where generated samples are saved |
| `-bootstrap` | off | run the bootstrap study |
| `-B` | 100 | bootstrap samples per study sample |
| `-iterations` | 100 | samples per sample size in the study |
| `-cpuprofile FILE`, `-memprofile FILE` | — | write pprof profiles |

Examples:

```bash
go run . -data mydata.csv -trim 0.1
go run . -data mydata.csv -lower 0.1 -upper 0
```

Invalid input produces a clear message and exit code 1, for example:

```
$ trimmedmean-demo -trim 0.7
error: trimmedmean: trimming proportion must be in [0, 0.5): got 0.7
run with -h to see the available options
```

Elapsed time and memory use are logged to stderr on every run, so stdout holds only results.

## Building an executable

```bash
go build -o trimmedmean-demo .                                          # this machine
GOOS=windows GOARCH=amd64 go build -o trimmedmean-demo.exe .            # Windows
GOOS=darwin  GOARCH=arm64 go build -o trimmedmean-demo-mac .            # macOS, Apple silicon
GOOS=darwin  GOARCH=amd64 go build -o trimmedmean-demo-mac-intel .      # macOS, Intel
```

Each is a single self-contained file (about 3 MB) with no dependencies. Run it from a
terminal (`trimmedmean-demo.exe -h` on Windows, `./trimmedmean-demo-mac -h` on macOS).
On macOS, if Gatekeeper blocks an unsigned binary, run
`xattr -d com.apple.quarantine trimmedmean-demo-mac` once.

### macOS .app bundle (optional)

This is a command-line program, so a plain executable is the natural form on macOS.
If you want a double-clickable `.app`, wrap the binary in a bundle that opens it in
Terminal:

```bash
GOOS=darwin GOARCH=arm64 go build -o trimmedmean-demo-mac .
mkdir -p TrimmedMeanDemo.app/Contents/MacOS
cp trimmedmean-demo-mac TrimmedMeanDemo.app/Contents/MacOS/trimmedmean-demo
cat > TrimmedMeanDemo.app/Contents/MacOS/launch <<'SH'
#!/bin/bash
DIR="$(cd "$(dirname "$0")" && pwd)"
open -a Terminal "$DIR/trimmedmean-demo"
SH
chmod +x TrimmedMeanDemo.app/Contents/MacOS/launch
cat > TrimmedMeanDemo.app/Contents/Info.plist <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleName</key><string>TrimmedMeanDemo</string>
  <key>CFBundleIdentifier</key><string>com.chbobray.trimmedmeandemo</string>
  <key>CFBundleExecutable</key><string>launch</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleVersion</key><string>0.1.1</string>
</dict></plist>
PLIST
```

Double-clicking `TrimmedMeanDemo.app` opens a Terminal window and runs the program with
its default settings; the generated `data/` folder is written to your home directory.

## Testing and profiling

```bash
go test -v -cover ./...                       # unit tests
go test -run=^$ -bench=. ./...                # benchmark of the bootstrap study
go run . -bootstrap -cpuprofile cpu.prof -memprofile mem.prof
go tool pprof -top cpu.prof
```

Tests (83% statement coverage) cover the median and standard deviation helpers (checked
against R's `median()` and `sd()`), exact CSV round-trips, every run mode (generated
samples, a user data file with asymmetric trimming, the bootstrap study with CPU and
memory profiles), rejection of invalid input (bad files, out-of-range trims, too few
resamples, unwritable profile paths), and reproducibility of the bootstrap study for a
fixed seed. The package itself has its own
tests in its repository.

## Files

| File | Role |
|---|---|
| `main.go` | command-line flags, input validation, the R-comparison run, profiling and logging |
| `data.go` | generating samples (contaminated normal mixture, integers with outliers), reading and writing CSV |
| `bootstrap.go` | the bootstrap study, plus `median` and `standardDeviation` helpers |
| `main_test.go` | unit tests of the helpers, and the bootstrap benchmark |
| `run_test.go` | tests of each run mode and of invalid-input handling |
| `compare.R` | R's `mean(x, trim = 0.05)` on the saved samples |
| `crosscheck.sh` | Go vs R on 80 samples of varying size |
| `run-bootstrap-robust.R` | the instructor's R study (Thomas W. Miller, 2024), for comparison |
| `data/` | the 100-integer and 100-float samples used above |
| `results/` | saved outputs from Go and R |

## GenAI Tools

This program was written with the help of Claude (Anthropic), an AI assistant. I gave it
the assignment description, the instructor's `run-bootstrap-robust.R` and its sample
listing. Claude designed and wrote the program, tests, comparison scripts, and README;
ran the Go program and R side by side to confirm the results match; ported the R
bootstrap study to Go and ran it with B = 100 and B = 1,000; and timed both versions.
I reviewed the code and results. The full conversation log is available at
[https://claude.ai/share/9448bf64-cd2d-49c9-b7c8-85bf1a67f41c](https://claude.ai/share/9448bf64-cd2d-49c9-b7c8-85bf1a67f41c).
