FROM ubuntu:16.04

RUN apt-get update
RUN apt-get install -y build-essential
RUN apt-get install -y clang
RUN apt-get install -y wget
RUN wget https://dl.google.com/go/go1.9.3.linux-amd64.tar.gz -O /tmp/go.tgz
RUN tar -C /usr/local -xf /tmp/go.tgz
RUN echo "export PATH=$PATH:/usr/local/go/bin" >> /root/.bashrc
COPY transform/target/libtransformd_linux_amd64.so /usr/local/lib/libtransformd.so
