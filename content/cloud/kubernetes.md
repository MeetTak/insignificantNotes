---
title: "kubernetes"
folder: "cloud"
created: "2026-08-15T14:55:30.000Z"
modified: "2026-08-27T15:48:15.000Z"
note_id: "x-coredata://8788A586-C8FC-4484-AB68-760AF85E0359/ICNote/p3774"
source: "apple-notes"
---

client server architecture

cluster pane

- kubeapi server
- etcd
- kubescheduler
- kube control manager
- cloud control manager

worker node

- kubelet
- kubeproxy
- container runtime (containerd)

data obj’s

- pods
- services

addons

- cluster dns
- ingress controller
- container network interface (CNI)

The kubelet is the fundamental "node agent" that runs on every single worker node inside a Kubernetes cluster
