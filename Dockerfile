# Stage 1
FROM alpine/git as clone
WORKDIR /
COPY . ./helios-core
# Stage 2
FROM golang:1.23.3-bullseye AS build
WORKDIR /helios-core
COPY --from=clone /helios-core/ .
RUN go build ./cmd/heliades/
# Stage 3
FROM node:18.16.0 AS final
WORKDIR /app
COPY --from=build /helios-core/heliades /usr/bin/heliades
COPY ./libwasmvm/libwasmvm.x86_64.so /lib/libwasmvm.x86_64.so
COPY ./libwasmvm/libwasmvm.aarch64.so /lib/libwasmvm.aarch64.so
ENTRYPOINT ["heliades", "version"]