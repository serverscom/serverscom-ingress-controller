FROM alpine:3.19

RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
COPY serverscom-ingress-controller /bin/serverscom-ingress-controller

ENTRYPOINT ["/bin/serverscom-ingress-controller"]
