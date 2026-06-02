FROM golang:1.26-bookworm AS build
WORKDIR /src
RUN apt-get update && apt-get install -y --no-install-recommends libopencv-dev pkg-config && rm -rf /var/lib/apt/lists/*
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go test -tags gocv ./internal/adapters/out/vision/local
RUN CGO_ENABLED=1 go build -tags gocv -o /server ./cmd/server

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends libopencv-dev ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=build /server /server
ENTRYPOINT ["/server"]
