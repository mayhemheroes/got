#!/usr/bin/env bash
#
# mayhem/test.sh — RUN got's own `go test ./...` suite (already compiled into the
# GOCACHE by mayhem/build.sh's warm-up pass). exit 0 = pass.
set -uo pipefail
: "${SRC:=/mayhem}"
[ -n "${SOURCE_DATE_EPOCH:-}" ] || unset SOURCE_DATE_EPOCH
: "${MAYHEM_JOBS:=$(nproc)}"
cd "$SRC"

emit_ctrf() {
  local tool="$1" passed="$2" failed="$3" skipped="${4:-0}" pending="${5:-0}" other="${6:-0}"
  local tests=$(( passed + failed + skipped + pending + other ))
  cat > "${CTRF_REPORT:-$SRC/ctrf-report.json}" <<JSON
{
  "results": {
    "tool": { "name": "$tool" },
    "summary": {
      "tests": $tests,
      "passed": $passed,
      "failed": $failed,
      "pending": $pending,
      "skipped": $skipped,
      "other": $other
    }
  }
}
JSON
  printf 'CTRF {"results":{"tool":{"name":"%s"},"summary":{"tests":%d,"passed":%d,"failed":%d,"pending":%d,"skipped":%d,"other":%d}}}\n' \
    "$tool" "$tests" "$passed" "$failed" "$pending" "$skipped" "$other"
  [ "$failed" -eq 0 ]
}

out="$(go test -count=1 -v ./... 2>&1)"
rc=$?
echo "$out"

passed=$(grep -c '^--- PASS:' <<<"$out")
failed=$(grep -c '^--- FAIL:' <<<"$out")
skipped=$(grep -c '^--- SKIP:' <<<"$out")

if [ "$passed" -eq 0 ] && [ "$failed" -eq 0 ] && [ "$skipped" -eq 0 ]; then
  # No per-test lines matched (e.g. a build failure) — fall back to the exit code.
  if [ "$rc" -eq 0 ]; then
    emit_ctrf "go-test" 1 0
  else
    emit_ctrf "go-test" 0 1
  fi
else
  emit_ctrf "go-test" "$passed" "$failed" "$skipped"
fi
