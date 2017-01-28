FROM alpine:3.5
MAINTAINER Jason Wieringa <jason@wieringa.io>

RUN apk --no-cache add \
        ca-certificates

ADD bin/ec2-sg-route53-linux-amd64 /opt/local/bin/ec2-sg-route53

CMD [ "/opt/local/bin/ec2-sg-route53" ]
