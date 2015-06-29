# etcdriver

This is a docker volume plugin for etcd. It uses [etcdfs](github.com/xetorthio/etcd-fs/src/etcdfs)
to mount a tree from etcd to a temporary path and passes this local path to docker. Inside your
container you can use normal filesystem functions to access this etcd subtree.

So you can have a etcd cluster and multiple docker containers which have their
configs in etcd stored without using the etcd rest API.

Well, mostly this is only a test plugin for me to learn the docker plugin API :-)

Usage:
```
docker run -it --rm --volume-driver=etcdriver -v @/config:/test ubuntu /bin/bash
```
to mount the subtree `/config` from your etcd into the docker container at
the path `/test`.