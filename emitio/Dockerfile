FROM ubuntu:16.04

RUN apt-get update
RUN apt-get install -y build-essential
RUN apt-get install -y clang
RUN apt-get install -y wget
RUN wget https://dl.google.com/go/go1.9.3.linux-amd64.tar.gz -O /tmp/go.tgz
RUN tar -C /usr/local -xf /tmp/go.tgz
RUN echo "export PATH=$PATH:/usr/local/go/bin" >> /root/.bashrc
RUN wget https://aka.ms/chakracore/cc_linux_x64_1_8_0 -O /tmp/chakra.tgz
RUN tar -xf /tmp/chakra.tgz
RUN cp /ChakraCoreFiles/lib/libChakraCore.so /usr/lib/