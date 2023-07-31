FROM messense/rust-musl-cross:x86_64-musl as builder

WORKDIR /app

ADD . .

RUN cargo build --release --target x86_64-unknown-linux-musl

FROM scratch

# We need SSL certs from the build server to connect to send https web requets 
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /app/target/x86_64-unknown-linux-musl/release/plex-ani-sync /app/

WORKDIR /app

ENTRYPOINT ["./plex-ani-sync"]
