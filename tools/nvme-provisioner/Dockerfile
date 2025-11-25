# Fix for issue 72
FROM public.ecr.aws/docker/library/debian:bookworm-slim

RUN  apt-get update && apt-get -y install nvme-cli mdadm && apt-get -y clean && apt-get -y autoremove
COPY nvme-provisioner.sh /usr/local/bin/

ENTRYPOINT ["nvme-provisioner.sh"]
