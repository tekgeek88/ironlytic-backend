# Stage 1: Build the Go binary
FROM golang:1.24.0 AS builder

# Set the environement
ARG ENV=staging
ENV ENV=${ENV}
WORKDIR /app

COPY . .

RUN make build ENV=$ENV

# Stage 2: Create minimal container
FROM alpine:latest
ARG ENV=staging
ENV ENV=${ENV}

WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /app/ironlytic-backend .

RUN apk add --no-cache ca-certificates

# Install tzdata for timezone support
RUN apk add --no-cache tzdata

# Fix permission just in case
RUN chmod +x ./ironlytic-backend

EXPOSE 8080
CMD ["./ironlytic-backend"]
