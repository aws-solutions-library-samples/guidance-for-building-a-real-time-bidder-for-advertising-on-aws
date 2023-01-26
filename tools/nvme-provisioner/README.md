nvme-provisioner
================

Slightly modified version of script from https://github.com/brunsgaard/eks-nvme-ssd-provisioner
The `eks-nvme-ssd-provisioner` Docker images cannot be used directly because they are outdated.

The scripts formats and mounts NVMe Instance Storage volumes. If there is more than one volume,
the script combines NMVe volumes in RAID0 device. The final device is mounted to `/nvme/disk` on the host.

To build Docker image:

```
make nvme-provisioner@build
```

To Docker image to the registry:

```
make nvme-provisioner@push
```
