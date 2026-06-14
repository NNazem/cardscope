FROM golang:1.26-bookworm AS go-toolchain

FROM ghcr.io/hybridgroup/opencv:4.13.0 AS build
COPY --from=go-toolchain /usr/local/go /opt/go
ENV GOROOT="/opt/go"
ENV PATH="/opt/go/bin:${PATH}"
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go test ./...
RUN CGO_ENABLED=1 go build -o /server ./cmd/server

FROM ghcr.io/hybridgroup/opencv:4.13.0
COPY --from=build /server /server
ENTRYPOINT ["/server"]
