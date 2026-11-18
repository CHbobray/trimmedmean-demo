#!/usr/bin/env bash
# Cross-check the Go trimmed mean against R's mean(x, trim = 0.05)
# on many generated samples of different sizes. Requires Go and Rscript.
#
# Usage: ./crosscheck.sh [number-of-seeds]
set -euo pipefail

SEEDS=${1:-40}
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

go build -o "$WORK/demo" .

for seed in $(seq 1 "$SEEDS"); do
    n=$((100 + seed * 37))                     # sizes 137, 174, ... all at least 100
    "$WORK/demo" -seed "$seed" -n "$n" -out "$WORK/s$seed" 2>/dev/null \
        | awk -v s="$seed" '/n = /{print s, $1, $NF}'
done > "$WORK/go.txt"

Rscript - "$WORK" <<'EOF'
work <- commandArgs(trailingOnly = TRUE)[1]
go <- read.table(file.path(work, "go.txt"), col.names = c("seed", "type", "go"))
go$file <- ifelse(go$type == "integers", "ints.csv", "floats.csv")
go$r <- mapply(function(seed, file)
                 mean(read.csv(file.path(work, paste0("s", seed), file))$x, trim = 0.05),
               go$seed, go$file)
diffs <- abs(go$go - round(go$r, 10))
cat(sprintf("Compared %d samples (%d integer, %d float), n from %d to %d\n",
            nrow(go), sum(go$type == "integers"), sum(go$type == "floats"),
            137, 100 + 37 * max(go$seed)))
cat(sprintf("Largest |Go - R| at 10 decimal places: %g\n", max(diffs)))
if (any(diffs > 0)) quit(status = 1)
cat("All Go results match R.\n")
EOF
