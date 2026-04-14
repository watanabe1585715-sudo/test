# 求人広告 API / バッチ用マルチステージビルド
# ビルド: docker build -t recruitment-api .
# 実行:  docker compose up api （CMD は API）
FROM golang:1.22-bookworm AS builder
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/api ./cmd/api \
 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/batch ./cmd/batch \
 && CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/mailworker ./cmd/mailworker

FROM debian:bookworm-slim
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates tzdata \
 && rm -rf /var/lib/apt/lists/*

ENV TZ=Asia/Tokyo
WORKDIR /app
COPY --from=builder /out/api /app/api
COPY --from=builder /out/batch /app/batch
COPY --from=builder /out/mailworker /app/mailworker

EXPOSE 8080
# 既定は API（compose の batch サービスで /app/batch を上書き実行）
CMD ["/app/api"]
