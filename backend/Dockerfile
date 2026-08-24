FROM golang:1.25-alpine AS build
WORKDIR /src
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache git ca-certificates
ENV GOPROXY=https://goproxy.cn,direct
ENV GOTOOLCHAIN=auto
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -o /out/storage ./cmd/storage \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/gateway ./cmd/gateway \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/agent ./cmd/agent \
    && CGO_ENABLED=0 GOOS=linux go build -o /out/loadgen ./cmd/loadgen

FROM alpine:3.20
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
    && apk add --no-cache ca-certificates wget tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY --from=build /out/storage /app/storage
COPY --from=build /out/gateway /app/gateway
COPY --from=build /out/agent /app/agent
COPY --from=build /out/loadgen /app/loadgen
EXPOSE 8080
HEALTHCHECK --interval=5s --timeout=3s --retries=12 CMD wget -qO- http://127.0.0.1:8080/health || exit 1
CMD ["/app/storage"]
