# Compare Go's trimmed mean with R's base mean(x, trim = 0.05).
# Reads the samples written by the Go program (data/ints.csv, data/floats.csv)
# so both languages work on exactly the same numbers.
#
# Usage: Rscript compare.R [data-directory]

args <- commandArgs(trailingOnly = TRUE)
data_dir <- if (length(args) > 0) args[1] else "data"
TRIM <- 0.05

cat("R trimmed mean results (trim", TRIM, ")\n")
for (name in c("ints", "floats")) {
  path <- file.path(data_dir, paste0(name, ".csv"))
  x <- read.csv(path)$x
  cat(sprintf("  %-8s %s  n = %d  mean = %.10f  trimmed mean = %.10f\n",
              ifelse(name == "ints", "integers", "floats"), path,
              length(x), mean(x), mean(x, trim = TRIM)))
}
