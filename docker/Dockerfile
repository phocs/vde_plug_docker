FROM debian:9.4-slim

RUN apt-get update
RUN apt-get install -y git make automake autoconf libtool

RUN cd /tmp/; git clone https://github.com/rd235/s2argv-execs.git
RUN cd /tmp/s2argv-execs/; autoreconf -if; ./configure; make; make install
RUN cd /tmp/; rm -rf s2argv-execs/; git clone https://github.com/rd235/vdeplug4.git;
RUN cd /tmp/vdeplug4; autoreconf -if; ./configure; make; make install; cd ..; rm -rf vdeplug4/

RUN apt-get purge -y git make automake autoconf; apt-get autoremove -y; apt autoclean -y

COPY ./vde_plug_docker /vde_plug_docker
ENV LD_LIBRARY_PATH=/usr/local/lib/
ENTRYPOINT ["/vde_plug_docker"]
