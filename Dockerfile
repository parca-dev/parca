FROM alpine:3.12

ARG ARCH=amd64

WORKDIR /conprof

COPY .build/linux-$ARCH/conprof /bin/conprof
COPY examples/conprof.yaml      /etc/conprof/config.yaml

RUN apk add --no-cache graphviz \
    && chown -R nobody:nogroup /etc/conprof /conprof

USER       nobody
EXPOSE     8080
ENTRYPOINT [ "/bin/conprof" ]
CMD        [ "all", \
             "--storage.tsdb.path=/conprof", \
             "--config.file=/etc/conprof/config.yaml" ]
