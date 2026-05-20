FROM node:22-alpine AS ui-build

WORKDIR /src/ui
COPY ui/package*.json ./
RUN npm ci
COPY ui ./
RUN npm run build

FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/worker-normalize ./cmd/worker-normalize \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/worker-transaction ./cmd/worker-transaction \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/worker-grouping ./cmd/worker-grouping \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/worker-alert ./cmd/worker-alert \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/worker-attachment ./cmd/worker-attachment \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/worker-profile ./cmd/worker-profile \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/worker-replay ./cmd/worker-replay \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/worker-session ./cmd/worker-session \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/worker-outcome ./cmd/worker-outcome \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/worker-event-writer ./cmd/worker-event-writer

FROM alpine:3.21

RUN adduser -D -H app
USER app

COPY --from=build /out/api /api
COPY --from=build /out/worker-normalize /worker-normalize
COPY --from=build /out/worker-transaction /worker-transaction
COPY --from=build /out/worker-grouping /worker-grouping
COPY --from=build /out/worker-alert /worker-alert
COPY --from=build /out/worker-attachment /worker-attachment
COPY --from=build /out/worker-profile /worker-profile
COPY --from=build /out/worker-replay /worker-replay
COPY --from=build /out/worker-session /worker-session
COPY --from=build /out/worker-outcome /worker-outcome
COPY --from=build /out/worker-event-writer /worker-event-writer
COPY --from=ui-build /src/ui/dist /ui
EXPOSE 8080

ENTRYPOINT ["/api"]
