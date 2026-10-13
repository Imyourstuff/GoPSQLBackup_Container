# BUILD APP WITH DEPS
FROM golang:alpine AS build-stage

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -ldflags="-w -s" -o /app/backup ./main.go


# DEPLOY BIN AND ASSETS
FROM alpine:latest as release-stage

WORKDIR /app

RUN apk --no-cache add postgresql-client bash tzdata findutils

COPY --from=build-stage /app/backup /app/backup

ENTRYPOINT [ "/app/backup" ]
