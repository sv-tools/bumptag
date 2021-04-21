FROM alpine

RUN apk update && apk upgrade && \
    apk add --no-cache git
COPY bumptag /usr/bin/bumptag

ENTRYPOINT [ "bumptag" ]
CMD ["--help"]
