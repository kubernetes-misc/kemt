FROM golang:1.13.7-alpine3.11 as vendor
RUN mkdir /user && \
    echo 'nobody:x:65534:65534:nobody:/:' > /user/passwd && \
    echo 'nobody:x:65534:' > /user/group
RUN apk add --no-cache ca-certificates git
WORKDIR /src
COPY go.mod ./
RUN go mod download
RUN go mod vendor


FROM vendor as builder
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app bin/app.go
RUN mkdir build
RUN mv app build/app
RUN mv -f views build/
RUN ls -la /src/build


#FROM scratch as final
#COPY --from=builder /user/group /user/passwd /etc/
#COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
#COPY --from=builder /src/build /build
#USER nobody:nobody
#ENTRYPOINT ["/build/app"]


FROM debian:stretch
RUN mkdir -p /build
WORKDIR /build
COPY --from=builder /src/build /build
ENTRYPOINT ["/build/app"]