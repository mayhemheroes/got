#!/usr/bin/env bash
#
# mayhem/build.sh — build got's go-fuzz harnesses as sanitized libFuzzer binaries
# (OSS-Fuzz Go path: go-fuzz-build -libfuzzer + clang link).
#
# Runs inside the commit image (mayhem/Dockerfile) as `mayhem` in /mayhem.
# GOROOT/GOPATH/GOMODCACHE are pinned by the Dockerfile ENV (under /opt/toolchains —
# absolute, $HOME-independent).
#
# AIR-GAPPED CONTRACT (SPEC §6.5): the PATCH tier re-runs THIS script OFFLINE.
#   - This FIRST build (in CI, online) populates the module cache under $GOMODCACHE.
#   - The module cache doubles as a FILE PROXY at $GOMODCACHE/cache/download. We set
#     GOPROXY to that file proxy FIRST, network LAST: the offline re-run resolves
#     entirely from the cache, and the network fallback only fills cache-misses on
#     this first online build. -mod=mod lets go-fuzz-build's `go get` update go.mod
#     from the cache. (GOPROXY=off is NOT enough — it blocks reading the version
#     list from the cache, which `go get` needs.)
set -euo pipefail

: "${SRC:=/mayhem}"
[ -n "${SOURCE_DATE_EPOCH:-}" ] || unset SOURCE_DATE_EPOCH

: "${CC:=clang}" ; : "${CXX:=clang++}" ; : "${LIB_FUZZING_ENGINE:=-fsanitize=fuzzer}"
# OSS-Fuzz Go path is ASan-only for the libFuzzer link (keep ASan regardless of base default).
: "${SANITIZER_FLAGS=-fsanitize=address}"
# DWARF < 4 (§6.2 item 10): clang's plain -g emits DWARF-5; force DWARF-3 on the link.
: "${GO_DEBUG_FLAGS:=-gdwarf-3}"
: "${MAYHEM_JOBS:=$(nproc)}"
export CC CXX LIB_FUZZING_ENGINE SANITIZER_FLAGS GO_DEBUG_FLAGS MAYHEM_JOBS

# Resolve modules offline-first from the in-image cache; network only as a fallback.
# $(go env GOMODCACHE) reads the pinned ENV, so it is correct under ANY $HOME.
export GOFLAGS="${GOFLAGS:--mod=mod}"
export GOPROXY="${GOPROXY:-file://$(go env GOMODCACHE)/cache/download,https://proxy.golang.org,direct}"

cd "$SRC"
go version

# got's dependencies (e.g. blobcache.io/blobcache) use generic type aliases
# (Go 1.24+). go-fuzz-build's bundled go/types predates that default, so it
# needs the GODEBUG opt-in explicitly or it fails to typecheck them.
export GODEBUG="${GODEBUG:+$GODEBUG,}gotypesalias=1"

# go-fuzz-build needs the go-fuzz-dep package and go-fuzz-headers on the module
# graph. With -mod=mod + the file-proxy GOPROXY this resolves from the cache offline
# (no-op once cached by the first online build).
go get github.com/dvyukov/go-fuzz/go-fuzz-dep
go get github.com/AdaLogics/go-fuzz-headers
go mod tidy

mkdir -p "$SRC/mayhem-build"

# DWARF-3 anchor (§6.2 item 10): this is a pure LINK of a prebuilt go-fuzz archive —
# clang has no source to apply $GO_DEBUG_FLAGS to, so its libFuzzer runtime's own
# DWARF-5 CU lands at .debug_info offset 0 regardless. Compile a tiny anchor object
# WITH -gdwarf-3 and link it FIRST so a DWARF-3 CU sits at offset 0 instead (what
# verify-repo's readelf reads). The anchor must define a symbol — an empty file
# emits no CU at all.
cat > "$SRC/mayhem-build/anchor.c" <<'EOF'
int mayhem_dwarf3_anchor(void) { return 0; }
EOF
$CC -g -gdwarf-3 -c "$SRC/mayhem-build/anchor.c" -o "$SRC/mayhem-build/anchor.o"

build_target() {
  local name="$1" harness_dir="$2"
  echo "=== building $name (go-fuzz-build -libfuzzer) ==="
  (
    cd "$SRC/$harness_dir"
    go-fuzz-build -libfuzzer -o "$SRC/mayhem-build/$name.a"
  )
  # Link the go-fuzz archive into a libFuzzer binary with clang (ASan). anchor.o
  # first so its DWARF-3 CU is the first .debug_info entry.
  $CXX $SANITIZER_FLAGS $GO_DEBUG_FLAGS $LIB_FUZZING_ENGINE \
    "$SRC/mayhem-build/anchor.o" "$SRC/mayhem-build/$name.a" -o "/mayhem/$name"
  echo "built /mayhem/$name"
}

build_target fuzz_got_branches mayhem/fuzz_got_branches
build_target fuzz_got_chunking mayhem/fuzz_got_chunking
build_target fuzz_got_gdat     mayhem/fuzz_got_gdat

# Pre-compile the project's own test suite (NORMAL flags, no sanitizer) so
# mayhem/test.sh only RUNS it — `go test -run=^$` compiles every test binary
# into $GOCACHE without executing any test, leaving test.sh a warm-cache `go test`.
echo "=== pre-compiling test suite ==="
go build ./...
go test -run=^$ -count=1 ./...

echo "build.sh complete"
