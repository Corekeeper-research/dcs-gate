FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download || true
COPY *.go ./
RUN CGO_ENABLED=0 go build -o /dcs-gate ./...

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=build /dcs-gate /app/dcs-gate
COPY data /app/data
EXPOSE 8080
CMD ["/app/dcs-gate"]
