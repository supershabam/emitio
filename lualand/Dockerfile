from ubuntu:xenial

COPY ./LuaJIT-2.0.5.tar.gz /tmp/

RUN apt-get update
RUN apt-get install -y build-essential
RUN tar -C /tmp -xf /tmp/LuaJIT-2.0.5.tar.gz
RUN cd /tmp/LuaJIT-2.0.5 && make

COPY ./musl-1.1.18.tar.gz /tmp/

RUN tar -C /tmp -xf /tmp/musl-1.1.18.tar.gz
RUN cd /tmp/musl-1.1.18 && ./configure && make install

RUN cd /tmp/LuaJIT-2.0.5 && make install

COPY ./go1.9.2.linux-amd64.tar.gz /tmp/
RUN tar -C /usr/local -xf /tmp/go1.9.2.linux-amd64.tar.gz

COPY bashrc /tmp/
RUN cat /tmp/bashrc >> ~/.bashrc
