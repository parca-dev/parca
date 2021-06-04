FROM alpine:3.12

ARG ARCH=amd64

COPY .build/linux-$ARCH/conprof /bin/conprof
COPY examples/conprof.yaml      /etc/conprof/config.yaml

RUN apk add --no-cache graphviz binutils \
    && mkdir -p /conprof \
    && chown -R nobody:nobody /etc/conprof /conprof

USER       nobody
EXPOSE     10902
VOLUME     [ "/conprof" ]
WORKDIR    /conprof
ENTRYPOINT [ "/bin/conprof" ]
CMD        [ "all", \
             "--storage.tsdb.path=/conprof", \
             "--config.file=/etc/conprof/config.yaml" ]
