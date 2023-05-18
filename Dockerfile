FROM alpine:latest

ADD bin/poddns-admission-webhook /poddns-admission-webhook
ENTRYPOINT ["/poddns-admission-webhook"]
#CMD ["/poddns-admission-webhook"]