FROM golang:1.24.0 AS build
WORKDIR /app
COPY go.mod ./
COPY go.sum ./
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /entrypoint

FROM scratch
COPY --from=build /entrypoint /entrypoint
USER 1000:1000

ENTRYPOINT ["/entrypoint"]
