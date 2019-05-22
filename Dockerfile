FROM  quay.io/prometheus/busybox:latest
LABEL authors="Frederic Branczyk"

COPY conprof                  /bin/conprof
COPY examples/conprof.yaml    /etc/conprof/config.yaml

RUN mkdir -p /conprof && \
    chown -R nobody:nogroup etc/conprof /conprof

USER       nobody
EXPOSE     8080
WORKDIR    /conprof
ENTRYPOINT [ "/bin/conprof" ]
CMD        [ "all", \
             "--storage.tsdb.path=/conprof", \
             "--config.file=/etc/conprof/config.yaml" ]
