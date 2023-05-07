FROM golang:1.19 as builder

COPY . /got
WORKDIR /got/mayhem

RUN go install github.com/dvyukov/go-fuzz/go-fuzz@latest github.com/dvyukov/go-fuzz/go-fuzz-build@latest
RUN go get github.com/dvyukov/go-fuzz/go-fuzz-dep
RUN go get github.com/AdaLogics/go-fuzz-headers
RUN apt update && apt install -y clang

RUN cd fuzz_got_branches && go-fuzz-build -libfuzzer -o fuzz_got_branches.a && \
    clang -fsanitize=fuzzer fuzz_got_branches.a -o fuzz_got_branches.libfuzzer

RUN cd fuzz_got_chunking && go-fuzz-build -libfuzzer -o fuzz_got_chunking.a && \
    clang -fsanitize=fuzzer fuzz_got_chunking.a -o fuzz_got_chunking.libfuzzer

RUN cd fuzz_got_gdat && go-fuzz-build -libfuzzer -o fuzz_got_gdat.a && \
    clang -fsanitize=fuzzer fuzz_got_gdat.a -o fuzz_got_gdat.libfuzzer

FROM debian:bookworm-slim
COPY --from=builder /got/mayhem/fuzz_got_branches/fuzz_got_branches.libfuzzer /
COPY --from=builder /got/mayhem/fuzz_got_chunking/fuzz_got_chunking.libfuzzer /
COPY --from=builder /got/mayhem/fuzz_got_gdat/fuzz_got_gdat.libfuzzer /