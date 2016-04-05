FROM golang:1.6
MAINTAINER paulcull <dev@pkhome.co.uk>

RUN apt-get -qy update && apt-get -qy install vim-common gcc mercurial supervisor

WORKDIR    /go/src/github.com/paulcull/go-webbrick
ADD        . /go/src/github.com/paulcull/go-webbrick

ADD etc/supervisor.conf /app/etc/supervisord.conf
  
RUN        go get -v

RUN  go build -ldflags " \
       -X main.buildVersion  $(grep "const Version " version.go | sed -E 's/.*"(.+)"$/\1/' ) \
       -X main.buildRevision $(git rev-parse --short HEAD) \
       -X main.buildBranch   $(git rev-parse --abbrev-ref HEAD) \
       -X main.buildDate     $(date +%Y%m%d-%H:%M:%S) \
       -X main.goVersion     $GOLANG_VERSION \
     "
