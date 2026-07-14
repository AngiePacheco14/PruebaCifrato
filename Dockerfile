# ---- build stage ----
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/cifrato ./cmd/cifrato

# ---- runtime stage ----
FROM alpine:3.20 AS runtime
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/cifrato /usr/local/bin/cifrato
EXPOSE 8080
ENTRYPOINT ["cifrato"]
CMD ["serve"]
