FROM debian:stretch-slim

RUN  apt-get update && apt-get -y install nvme-cli mdadm && apt-get -y clean && apt-get -y autoremove
COPY nvme-provisioner.sh /usr/local/bin/

ENTRYPOINT ["nvme-provisioner.sh"]
