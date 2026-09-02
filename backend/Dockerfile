FROM golang:1.26.5-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM alpine:3.22 AS runtime

RUN apk --no-cache add ca-certificates \
    && adduser -D -u 10001 app

WORKDIR /app

# --chown on the COPY, not a later RUN chown: rewriting the binary's ownership
# in its own layer stores a second full copy of it in the image.
COPY --from=builder --chown=app:app /out/server /app/server

# The server writes uploads here and runs as a non-root user, so the directory
# has to exist and be owned by that user before it drops privileges.
RUN mkdir -p /app/uploads && chown app:app /app/uploads

USER app

EXPOSE 8080

CMD ["/app/server"]

# A one-shot image that applies migrations, so a fresh deployment has a schema.
# Separate from the server image on purpose: goose and the .sql files have no
# business in the running container.
FROM golang:1.26.5-alpine AS goose-build
RUN go install github.com/pressly/goose/v3/cmd/goose@v3.27.2

FROM alpine:3.22 AS migrator
COPY --from=goose-build /go/bin/goose /usr/local/bin/goose
COPY sql/migrations /migrations
ENV GOOSE_DRIVER=postgres
ENV GOOSE_MIGRATION_DIR=/migrations
ENTRYPOINT ["goose"]
CMD ["up"]
