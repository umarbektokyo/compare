# Stage 1: Build frontend
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go binary
FROM golang:1.25-alpine AS backend
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/*.go ./
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o compare-server .

# Stage 3: Final minimal image
FROM alpine:3.21
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend /app/compare-server .
COPY --from=frontend /app/frontend/build ./frontend/build
EXPOSE 8080
VOLUME /app/data
ENV PORT=8080
ENV DATA_DIR=/app/data
CMD ["./compare-server"]
