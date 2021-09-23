FROM golang:alpine3.14 as BUILDER
WORKDIR /opt/zos
COPY . .
WORKDIR /opt/zos/tools/subproxy/
RUN go build

FROM alpine:3.13.5
COPY --from=BUILDER /opt/zos/tools/subproxy/subproxy /bin/
ENTRYPOINT [ "/bin/subproxy" ]