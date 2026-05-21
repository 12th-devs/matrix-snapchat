FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates tzdata libolm3 \
    && rm -rf /var/lib/apt/lists/*

COPY bin/mautrix-snapchat-bridgev2 /usr/local/bin/mautrix-snapchat-bridgev2

WORKDIR /data
EXPOSE 4000
ENTRYPOINT ["/usr/local/bin/mautrix-snapchat-bridgev2"]
