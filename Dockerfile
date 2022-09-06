FROM ubuntu

RUN apt-get update
RUN apt-get install -y apt-utils 
RUN apt-get install -y curl vim less
RUN apt-get install -y ca-cacert ca-certificates

# copy selectively to avoid including secrets
WORKDIR /data 
COPY gatherbot /
COPY loop.sh /
COPY sync.sh /

CMD /data/loop.sh
