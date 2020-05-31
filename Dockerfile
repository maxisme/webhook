FROM golang:alpine AS builder
ADD . /app/
WORKDIR /app
RUN go build -o app

FROM alpine
WORKDIR /app
RUN touch config.json
COPY --from=builder /app/app /app/app
RUN apk add curl
HEALTHCHECK CMD curl --fail http://localhost:8080/health || exit 1
CMD ["./app"]