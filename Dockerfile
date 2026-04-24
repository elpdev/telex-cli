FROM golang:1.26.2 AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /out/telex ./cmd/telex

FROM gcr.io/distroless/static-debian12

COPY --from=build /out/telex /telex

ENTRYPOINT ["/telex"]
