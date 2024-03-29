# Create the SDK layers
FROM alpine/git as builder

WORKDIR /

# Fetch Tebis
RUN git clone --branch master "https://tebis-docker-container:kEmvUT1ZaceUsad6usGF@carvgit.ics.forth.gr/storage/tebis.git"


# Create the manager container
FROM python:3.6-alpine as tebis-manager

# Install Zookeeper dependency
RUN pip3 install kazoo

WORKDIR /
COPY --from=builder /tebis/scripts/kreonR/ .


ENTRYPOINT ["python"]
CMD ["/tebis_zk_init.py"]
