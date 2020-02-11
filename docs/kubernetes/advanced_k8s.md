# Advanced kubernetes features

## Storage

### Local Path k3s storage class

When deploying an application that needs to retain data, you’ll need to create persistent storage. Persistent storage allows you to store application data external from the pod running your application. This storage practice allows you to maintain application data, even if the application’s pod fails.

A persistent volume (PV) is a piece of storage in the Kubernetes cluster, while a persistent volume claim (PVC) is a request for storage.

![Persistant storage in kubernetes](ressources/storage/simple-localpath/persistentstorage.png)

PV has three access modes

- RWO: Read Write Once. It can only be read/write on one node at any given time
- RWX: Read Write Many. It can only be read/write on multiple node at the same time
- ROX: Read Only Many

K3s comes with Rancher’s Local Path Provisioner and this enables the ability to create persistent volume claims out of the box using local storage on the respective node.
StorageClass "local-path": Only support ReadWriteOnce access mode

let's create a hostPath backed persistent volume claim and a pod to utilize it:

```
$ cd ressources/storage/simple-localpath/
$ kubectl create -f pvc.yaml
$ kubectl create -f pod.yaml
$ kubectl get pods -n default
NAME                         READY   STATUS    RESTARTS   AGE
volume-test                  1/1     Running   0          2m

```

### Rook

Rook is an open source cloud-native storage orchestrator, providing the platform, framework, and support for a diverse set of storage solutions to natively integrate with cloud-native environments.

Rook turns storage software into self-managing, self-scaling, and self-healing storage services. It does this by automating deployment, bootstrapping, configuration, provisioning, scaling, upgrading, migration, disaster recovery, monitoring, and resource management. Rook uses the facilities provided by the underlying cloud-native container management, scheduling and orchestration platform to perform its duties.

installing [rook](https://rook.io/docs/rook/v1.2/)

```
git clone --single-branch --branch release-1.2 https://github.com/rook/rook.git
cd cluster/examples/kubernetes/ceph
kubectl create -f common.yaml
kubectl create -f operator.yaml
kubectl create -f cluster-test.yaml
```

### Installing [rook NFS](https://rook.io/docs/rook/v1.2/nfs.html)

NFS allows remote hosts to mount filesystems over a network and interact with those filesystems as though they are mounted locally. This enables system administrators to consolidate resources onto centralized servers on the network.

#### First deploy the Rook NFS operator using the following commands:

```
$ cd resources/storage/rook-NFS
$ kubectl create -f 1-operator.yaml
```

We will create a NFS server instance that exports storage that is backed by the default StorageClass. In k3s environments storageClass "local-path": Only support ReadWriteOnce access mode so for the PVC taht must be created before creating NFS CRD instance.

```
$ kubectl create -f 2-nfs.yaml
$ kubectl get pvc -n rook-nfs
NAME                STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
nfs-default-claim   Bound    pvc-9804c4f1-80e1-45ec-bf05-3dbdd012564e   1Gi        RWO            local-path     3m49s
$ kubectl get po -n rook-nfs
NAME         READY   STATUS    RESTARTS   AGE
rook-nfs-0   1/1     Running   0          16s
```

#### Accessing the Export through dynamic NFS provisioning

Once the NFS Operator and an instance of NFSServer is deployed. A storageclass has to be created to dynamically provisioning volumes.
The StorageClass need to have the following 3 parameters passed.

- exportName: It tells the provisioner which export to use for provisioning the volumes.
- nfsServerName: It is the name of the NFSServer instance.
- nfsServerNamespace: It namespace where the NFSServer instance is running.

```
$ kubectl create -f 3-sc.yaml
```

Once the above storageclass has been created create a PV claim referencing the storageclass as shown in the example given below.

```
$ kubectl create -f 4-pvc.yaml
```

#### Consuming the Export

Now we can consume the PV that we just created by creating an example web server app that uses the above `PersistentVolumeClaim` to claim the exported volume. There are 2 pods that comprise this example:

- A web server pod that will read and display the contents of the NFS share
- A writer pod that will write random data to the NFS share so the website will continually update
  Start both the busybox pod (writer) and the web server from the ressources/storage/rook-NFS folder:

```
kubectl create -f busybox-rc.yaml
kubectl create -f web-rc.yaml
```

CANT CREATE POD SUCK IN CONTAINER CREATING

```
  Warning  FailedMount       15m                   kubelet, zv2k8s-04  Unable to attach or mount volumes: unmounted volumes=[rook-nfs-vol], unattached volumes=[default-token-f6sx2 rook-nfs-vol]: timed out waiting for the condition
  Warning  FailedMount       2m31s (x10 over 16m)  kubelet, zv2k8s-04  MountVolume.SetUp failed for volume "pvc-903d405d-0c9c-4af7-bc2d-356fc03905fb" : mount failed: exit status 255
Mounting command: mount
Mounting arguments: -t nfs 10.43.102.171:/nfs-default-claim /var/lib/kubelet/pods/91d477ac-7e3f-4957-b427-0a3f2a68847b/volumes/kubernetes.io~nfs/pvc-903d405d-0c9c-4af7-bc2d-356fc03905fb
```

probably need nfs-common

### Installing [rook CEPH](https://rook.io/docs/rook/v1.2/ceph.html)

RBD

Rook Ceph requires a Linux kernel built with the RBD module. Many distributions of Linux have this module but some don’t, e.g. the GKE Container-Optimised OS (COS) does not have RBD. You can test your Kubernetes nodes by running modprobe rbd. If it says ‘not found’, you may have to rebuild your kernel or choose a different Linux distribution.

CephFS

If you will be creating volumes from a Ceph shared file system (CephFS), the recommended minimum kernel version is 4.17. If you have a kernel version less than 4.17, the requested PVC sizes will not be enforced. Storage quotas will only be enforced on newer kernels.

No RDB module and kernel < 4.17

```
usr/lib/modules/4.14.82-Zero-OS/kernel/drivers # find /usr/lib/modules | grep rdb
/usr/lib/modules/4.14.82-Zero-OS/kernel/drivers/media/rc/keymaps/rc-avermedia-cardbus.ko
```

## Load balancing and external IP

#### Klipper

K3s comes with [klipper](https://github.com/rancher/klipper-lb). It assigns the IP of the node to the loadBalancer service. This works by using a host port for each service load balancer and setting up iptables to forward the request to the cluster IP.

This means that klipper can't provision other IP for that we need metalLB

#### Metal LB

The idea is to have a fix IP to connect to that can handle load balancing between the pods. We use metallb to provision IP on demand for LoadBalancer services.
We can apply the first two files in the metallb folder

```
$ cd ressources/metallb
$ kubectl create -f 1-metallb.yaml
$ kubectl create -f 2-config_and_registration_service.yaml
```

this will install metallb on our cluster and configure it via a configmap
and lastly it will create a service of type loadbalancer with a fix IP `173.30.1.168`
Take a look at the configmap if you want to modify the IP range available to metalLB.

## Public IP

### Accessing the cluster from a public domain

we need to initiate a connection from the cluster to an external machine with a public IP. We direct the tunnel to the traefik (or any other ingress controller) service IP

```
$ kubectl get svc -n kube-system
NAME             TYPE           CLUSTER-IP      EXTERNAL-IP   PORT(S)                                     AGE
kube-dns         ClusterIP      10.43.0.10      <none>        53/UDP,53/TCP,9153/TCP                      6d5h
metrics-server   ClusterIP      10.43.36.33     <none>        443/TCP                                     6d5h
traefik          LoadBalancer   10.43.178.254   172.31.1.50   80:32015/TCP,443:32432/TCP,8080:30137/TCP   6d5h
```

We tried with [inlets](https://github.com/inlets/inlets).

Follow the guide and you will end up executing this kind of command on your master node.

```
inlets client  --remote "mydomain.be:8080"  --upstream "http://172.30.1.50:80"  --token "${AUTHTOKEN}"
```

If you have deployed the drupal-mysql example which contains an ingress ressources you can modify the ingress to the domain name here it is mydomain.be so that now when you hit http://mydomain.be you are redirected to the drupal website

## CNI

Container networking is the mechanism through which containers can optionally connect to other containers, the host, and outside networks like the internet.
The idea behind the CNI initiative is to create a framework for dynamically configuring the appropriate network configuration and resources when containers are provisioned or destroyed.

### Benchmark

**Ressources consumption**

![CNI Ressources consumption](ressources/cni/ressource_consumption.png)

**Performance**

![CNI Performance](ressources/cni/CNIPerf.png)

**Network Policies and Encryption**

![Network Policies and Encryption](ressources/cni/networkpolicies_encryption.png)

### Flannel by default

It is one of the most mature examples of networking fabric for container orchestration systems, intended to allow for better inter-container and inter-host networking.

From an administrative perspective, it offers a simple networking model that sets up an environment that’s suitable for most use cases when you only need the basics.

By default, K3s will run with flannel as the CNI.
The default backend for flannel is VXLAN. We can enable encryption by passing the IPSec (Internet Protocol Security) or WireGuard options .

### Calico

Project Calico, or just Calico, is another popular networking option in the Kubernetes ecosystem. While Flannel is positioned as the simple choice, Calico is best known for its performance, flexibility, and power. Calico takes a more holistic view of networking, concerning itself not only with providing network connectivity between hosts and pods, but also with network security and administration. The Calico CNI plugin wraps Calico functionality within the CNI framework.

calico kernel requirement

- nf_conntrack_netlink subsystem
- ip_tables (for IPv4)
- ip6_tables (for IPv6)
- ip_set
- xt_set
- ipt_set
- ipt_rpfilter
- ipt_REJECT
- ipip (if using Calico networking)

Unlike Flannel, Calico does not use an overlay network. Instead, Calico configures a layer 3 network that uses the BGP routing protocol to route packets between hosts. This means that packets do not need to be wrapped in an extra layer of encapsulation when moving between hosts. The BGP routing mechanism can direct packets natively without an extra step of wrapping traffic in an additional layer of traffic.

In addition to networking connectivity, Calico is well-known for its advanced network features. Network policy is one of its most sought after capabilities. In addition, Calico can also integrate with Istio, a service mesh, to interpret and enforce policy for workloads within the cluster both at the service mesh layer and the network infrastructure layer. This means that you can configure powerful rules describing how pods should be able to send and accept traffic, improving security and control over your networking environment.

Run K3s with --flannel-backend=none
./k3s server --flannel-backend=none &

## HA :warning: (WIP) :construction_worker:

![k3s ha architecture](ressources/ha/k3s-ha-architecture.svg)
