FROM alpine

RUN apk add --update ca-certificates

COPY gopath/bin/eventdb /eventdb

CMD ["/eventdb"]
