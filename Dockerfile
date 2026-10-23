FROM golang:1.26 AS build

WORKDIR /go/src/github.com/atiratree/etcd-backup
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/etcd-backup-plugin .

FROM registry.access.redhat.com/ubi9-minimal
COPY --from=build /go/bin/etcd-backup-plugin /plugins/
ENTRYPOINT ["cp", "/plugins/etcd-backup-plugin", "/target/."]
