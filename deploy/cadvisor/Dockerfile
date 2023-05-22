FROM  gcr.io/cadvisor/cadvisor:v0.46.0

# Needed to fix DNS issues
#COPY /etc/resolv.conf /etc/resolv.conf.override
#RUN echo "$( cat /etc/resolv.conf | sed 's/ndots:5/ndots:1/')"  >  /etc/resolv.conf

# Add a newer version of nsenter that supports cgroup
RUN apk add --no-cache util-linux inotify-tools
