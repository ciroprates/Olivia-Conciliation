# Build stage
FROM golang:1.24.11 AS build

WORKDIR /src

# Cache deps first
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY backend ./backend

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/olivia-backend ./backend

# Runtime stage
FROM gcr.io/distroless/base-debian12

WORKDIR /app
COPY --from=build /out/olivia-backend /app/olivia-backend

ENV PORT=8080
EXPOSE 8080

USER nonroot:nonroot
ENTRYPOINT ["/app/olivia-backend"]
