FROM alpine:3.8

RUN mkdir /app
WORKDIR /app

COPY . /app
RUN ["chmod", "+x","/app/sidecar-inject-server"]

ENTRYPOINT ["/app/sidecar-inject-server"]