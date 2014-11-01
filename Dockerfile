FROM google/golang:1.3

WORKDIR /gopath/src/app
ADD . /gopath/src/app/
RUN go get app

CMD []
EXPOSE 8090
ENTRYPOINT ["/gopath/bin/app"]