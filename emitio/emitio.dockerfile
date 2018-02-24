FROM ubuntu:16.04

COPY target/emitio_linux_amd64 /usr/local/bin/emitio
COPY transform/target/libtransformd_linux_amd64.so /usr/local/lib/libtransformd.so
RUN ldconfig \
    && apt-get update \
    && apt-get install -y ca-certificates \
    && rm -rf /var/lib/apt/lists/*

ENTRYPOINT [ "/usr/local/bin/emitio" ]
