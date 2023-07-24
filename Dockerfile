FROM rust:latest as builder

WORKDIR /app

ADD . .

RUN cargo build --release

FROM rust:latest

COPY --from=builder /app/target/release/plex-ani-sync ./

ENTRYPOINT ["./plex-ani-sync"]
