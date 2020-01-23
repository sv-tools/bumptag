FROM alpine:latest

RUN apk add --no-cache git
COPY bumptag /usr/bin/bumptag

ENTRYPOINT [ "bumptag" ]
CMD ["--help"]
