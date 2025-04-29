FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /pr-kind-labeler ./main.go

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /pr-kind-labeler /usr/local/bin/pr-kind-labeler
ENTRYPOINT ["/usr/local/bin/pr-kind-labeler"]
