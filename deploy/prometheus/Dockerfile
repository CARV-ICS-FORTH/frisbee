FROM prom/prometheus

USER root

# Use envsubst to replace ${var} or $var according to the values of the current environment variables.
RUN wget -O /bin/envsubst https://github.com/a8m/envsubst/releases/download/v1.2.0/envsubst-`uname -s`-`uname -m`

RUN chmod +X /bin/envsubst && chmod 777 /bin/envsubst

USER nobody
WORKDIR /prometheus
