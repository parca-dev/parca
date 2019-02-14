FROM  quay.io/prometheus/busybox:latest
LABEL authors="Frederic Branczyk, Goutham Veeramachaneni <gouthamve@gmail.com>"

COPY conprof                  /bin/conprof
COPY examples/conprof.yaml    /etc/conprof/config.yaml

EXPOSE     8080 
ENTRYPOINT [ "/bin/conprof" ]
CMD        [ "all", "--config.file=/etc/conprof/config.yaml" ]
