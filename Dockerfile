FROM golang:1.6
MAINTAINER paulcull <dev@pkhome.co.uk>

RUN apt-get -qy update && apt-get -qy install vim-common gcc mercurial supervisor

WORKDIR    /go/src/github.com/paulcull/go-webbrick
ADD        . /go/src/github.com/paulcull/go-webbrick

ADD etc/supervisor.conf /etc/supervisor/conf.d/go-webbrick.conf
  
RUN  go get -v

RUN  go build -ldflags " \
       -X main.buildVersion  $(grep "const Version " version.go | sed -E 's/.*"(.+)"$/\1/' ) \
       -X main.buildRevision $(git rev-parse --short HEAD) \
       -X main.buildBranch   $(git rev-parse --abbrev-ref HEAD) \
       -X main.buildDate     $(date +%Y%m%d-%H:%M:%S) \
       -X main.goVersion     $GOLANG_VERSION \
     "

EXPOSE 9001
#RUN supervisord -c /etc/supervisor/supervisord.conf
CMD ["/usr/bin/supervisord -c /etc/supervisor/conf.d/supervisord.conf"]
