FROM golang:1.25-alpine AS builder

WORKDIR /project
ADD go.* ./
RUN go mod download
ADD . .
RUN CGO_ENABLED=0 go build -o service ./cmd/service/main.go

FROM surnet/alpine-wkhtmltopdf:3.17.0-0.12.6-full

RUN apk add --no-cache ttf-dejavu font-noto-cjk font-noto-cjk-extra

WORKDIR /project
COPY --from=builder /project/service service
ENV UI_THEME=dark
ENTRYPOINT ["./service"]
