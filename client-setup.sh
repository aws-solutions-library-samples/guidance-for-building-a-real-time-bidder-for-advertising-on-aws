#!/usr/bin/env bash
set -ex

# download specific version of helm
curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
chmod 700 get_helm.sh && ./get_helm.sh --version v3.16.1

# download kubectl
#curl -LO "https://dl.k8s.io/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl.sha256"
curl -LO "https://dl.k8s.io/release/v1.30.0/bin/linux/amd64/kubectl"

# install jq and helm
# check for os
if [ "$(uname)" == "Linux" ]; then
    sudo yum install -y jq
    sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
elif [ "$(uname)" == "Darwin" ]; then
    brew install jq
    brew install kubectl
fi
# install kubectl specific version

kubectl version --client --output=yaml    

