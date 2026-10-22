FROM golang:1.27-alpine AS base

WORKDIR /app
RUN apk add --no-cache git
COPY backend/go.mod backend/go.sum ./
RUN go mod download

FROM base AS development
RUN go install github.com/air-verse/air@v1.61.7
COPY backend/.air.toml ./.air.toml
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

FROM base AS build
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot AS production
COPY --from=build --chown=nonroot:nonroot /out/server /server
EXPOSE 8080
ENTRYPOINT ["/server"]
