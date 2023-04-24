FROM golang:1.17-alpine

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o /server .

FROM alpine

COPY --from=0 /server /

CMD ["/server"]
