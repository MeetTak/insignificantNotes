---
title: "docker"
folder: "cloud"
created: "2026-08-15T15:15:39.000Z"
modified: "2026-08-16T14:55:12.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p3775"
source: "apple-notes"
---

A Docker image has multiple read-only layers. This means the image is built in stacked parts. Each part saves changes from the step before it. You cannot change these layers. When you run a container, Docker adds a thin read-write layer on top.

imp comparisons
cmd and run
cmd and entrypoint
copy and add

The following are the methods by which we can achieve docker image optimization.

1. Using distroless/minimal base images
2. Multistage builds
3. Minimizing the number of layers
4. Understanding caching
5. Using Dockerignore
6. Keeping application data elsewhere

docker is compatible only for linux, why:
namespaces
cgroups
union filesystems: helps to layer filesystems and provide efficient image layering, which helps in creating lightweight and shareable container images.
Docker also provides solutions like Docker Desktop for Mac and Windows that run a lightweight Linux VM to enable Docker usage on non-Linux platforms, effectively providing a Linux environment for Docker to operate within.
