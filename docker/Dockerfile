ARG GO_VERSION=1.24
ARG TARGETOS=linux
ARG TARGETARCH=amd64

FROM --platform=${TARGETOS}/${TARGETARCH} golang:${GO_VERSION} AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG SERVICE
RUN test -n "$SERVICE"

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -o /out/app ./cmd/${SERVICE}

FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=build /out/app /app/app
USER nonroot:nonroot

ENTRYPOINT ["/app/app"]