#!/usr/bin/env bash
set -e
set -x

if [ $# -ne 4 ]; then
  echo "Usage: ${0} registry_url architectures image_list"
  echo
  exit 1
fi

imageprefix=${1}
registry=${2}
architectures=${3}
images=${4}

for image in ${images}; do
  arch_images=()
  arch_annotations=()

  repository=$(echo ${image} | cut -d ':' -f 1 | cut -d '/' -f 2)
  tag=$(echo ${image} | cut -d ':' -f 2)
  target_image="${registry}/${imageprefix}${repository}:${tag}"

  docker --version
  
  # remove existing manifest
   docker manifest rm "${target_image}" || true

  # mirror the image for specific platform
  for arch in ${architectures}; do
    arch_tag="${tag}-${arch}"
    arch_image="${registry}/${imageprefix}${repository}:${arch_tag}"

    arch_images+=("${arch_image}")
    arch_annotations+=("--arch ${arch} ${target_image} $arch_image")

#    docker pull "${image}"
    docker pull "--platform=${arch}" "${image}"
    docker tag "${image}" "${arch_image}"
    docker push "${arch_image}"
  done

  # create manifest
  docker manifest create "${target_image}" "${arch_images[@]}"

  # annotate image variants in the manifest
  for annotation in "${arch_annotations[@]}"; do
    docker manifest annotate ${annotation}
  done

  # push the final manifest
  docker manifest push "${target_image}"
done
