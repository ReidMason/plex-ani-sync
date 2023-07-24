FROM messense/rust-musl-cross:x86_64-musl as builder

WORKDIR /app

ADD . .

RUN cargo build --release --target x86_64-unknown-linux-musl

FROM scratch

COPY --from=builder /app/target/x86_64-unknown-linux-musl/release/plex-ani-sync  ./

ENTRYPOINT ["./plex-ani-sync"]
