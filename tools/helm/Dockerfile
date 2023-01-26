ARG REGISTRY
FROM ${REGISTRY}/debian:stretch-slim

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y apt-transport-https apt-utils ca-certificates curl gnupg
RUN curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg
RUN echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | tee /etc/apt/sources.list.d/kubernetes.list
RUN apt-get update -o Dir::Etc::sourcelist="/etc/apt/sources.list.d/kubernetes.list" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" && apt-get install -y kubectl

RUN curl https://baltocdn.com/helm/signing.asc | apt-key add -
RUN echo "deb https://baltocdn.com/helm/stable/debian/ all main" | tee /etc/apt/sources.list.d/helm-stable-debian.list
RUN apt-get update -o Dir::Etc::sourcelist="/etc/apt/sources.list.d/helm-stable-debian.list" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0" && apt-get install -y helm
