FROM alpine
ADD ./sitemapper /
RUN apk update
RUN apk add ca-certificates
RUN rm -rf /var/cache/apk/*
ENTRYPOINT ["/sitemapper"]
