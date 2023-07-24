FROM rust:latest AS builder

WORKDIR /app

copy . .

RUN cargo build --release

RUN pwd

# FROM rust:latest
FROM debian:buster-slim

COPY --from=builder /app/target/release/plex-ani-sync ./

CMD ["ls"]

CMD ["./plex-ani-sync"]
