FROM golang:1.22-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /bin/backend ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=build /bin/backend /app/backend
COPY db /app/db
EXPOSE 8080
ENV PORT=8080
CMD ["/app/backend"]
