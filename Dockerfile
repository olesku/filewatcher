FROM golang:1.15-alpine as builder

RUN mkdir -p /build
WORKDIR /build


COPY . .
COPY go.mod .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o filewatcher .

FROM scratch
COPY --from=builder /build/filewatcher /
ENTRYPOINT [ "/filewatcher" ]
