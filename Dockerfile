#install packages for build layer
FROM golang:1.22.4-bookworm as builder
RUN apt install git gcc make libc-dev

ADD https://github.com/CosmWasm/wasmvm/releases/download/v2.1.2/libwasmvm.x86_64.so /lib/libwasmvm.x86_64.so
ADD https://github.com/CosmWasm/wasmvm/releases/download/v2.1.2/libwasmvm.aarch64.so /lib/libwasmvm.aarch64.so

#build binary
WORKDIR /src
COPY go.mod .
COPY go.sum .
ENV GO111MODULE=on
RUN go mod download
COPY . .

#build binary
RUN LEDGER_ENABLED=false make install-ci

# RUN LEDGER_ENABLED=false make install-evmos

#install gex
# RUN go install github.com/cosmos/gex@latest

#build main container
FROM debian:bookworm-slim
COPY --from=builder /go/bin/* /usr/local/bin/
COPY --from=builder /src/heliades.sh .
COPY --from=builder /src/setup.sh .

RUN apt update && apt install -y curl lz4 wget procps jq

RUN apt-get clean && apt-get autoclean && apt-get autoremove && rm -rf /var/lib/apt/lists/\* /tmp/\* /var/tmp/*

ADD https://github.com/CosmWasm/wasmvm/releases/download/v2.1.2/libwasmvm.x86_64.so /lib/libwasmvm.x86_64.so
ADD https://github.com/CosmWasm/wasmvm/releases/download/v2.1.2/libwasmvm.aarch64.so /lib/libwasmvm.aarch64.so

#configure container
VOLUME /apps/data
WORKDIR /apps/data
EXPOSE 26657 26656 10337 9900 9091 9999 1317 8545

COPY --from=builder /src/genesis.json /root/.heliades/config/genesis.json

RUN bash /setup.sh
#default command
CMD sh /heliades.sh
