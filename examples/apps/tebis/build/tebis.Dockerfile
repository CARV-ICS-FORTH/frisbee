# Create the SDK layers
ARG CENTOS=7.7.1908
FROM centos:${CENTOS} as builder

WORKDIR /build

# Install global tools
RUN yum groupinstall -y "Development tools"

# Install Zookeeper dependencies
RUN yum install -y ant automake cppunit-devel wget

# Install Zookeeper

RUN wget https://archive.apache.org/dist/zookeeper/zookeeper-3.5.9/apache-zookeeper-3.5.9.tar.gz && \
    tar xzf apache-zookeeper-3.5.9.tar.gz && \
    (cd apache-zookeeper-3.5.9 && ant compile_jute) && \
    (cd apache-zookeeper-3.5.9/zookeeper-client/zookeeper-client-c && autoreconf -if && ./configure && make install) && \
    rm -rf apache-zookeeper-3.5.9.tar.gz apache-zookeeper-3.5.9 && \
    ldconfig /usr/local/lib


# Install Tebis Dependencies
RUN yum install -y epel-release centos-release-scl
RUN yum install -y cmake3 devtoolset-10 boost-devel scl-utils

RUN yum groupinstall -y "Infiniband Support"
RUN yum install -y numactl-libs  numactl-devel libibverbs-devel  librdmacm infiniband-diags
RUN yum -y install perftest gperf


COPY CMakeLists.txt /root/CMakeLists.txt

# Install Tebis
RUN git clone --branch master "https://tebis-docker-container:kEmvUT1ZaceUsad6usGF@carvgit.ics.forth.gr/storage/tebis.git" tebis && \
    mkdir tebis/build && \
    cp /root/CMakeLists.txt ./tebis && \
    (cd tebis/build && scl enable devtoolset-10 -- /bin/bash -c "cmake3 -DCMAKE_BUILD_TYPE=\"Debug\" -DBUILD_SHARED_LIBS=OFF .. && make -j8")


RUN yum install -y ethtool librdmacm-utils

# Create the manager container
#FROM  centos:${CENTOS} as tebis-node

#RUN  yum install -y numactl-libs  numactl-devel libibverbs librdmacm
# yum groupinstall -y "Infiniband Support"
# yum -y install infiniband-diags perftest gperf

#WORKDIR /
#COPY --from=builder /root/tebis/build/kreon_server ./kreon-server

#ENTRYPOINT ["/kreon-server/kreon_server"]
