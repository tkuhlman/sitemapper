FROM alpine
ADD ./sitemapper /
ADD ./webroot /webroot
RUN apk update
RUN apk add ca-certificates
RUN rm -rf /var/cache/apk/*
ENTRYPOINT ["/sitemapper"]
