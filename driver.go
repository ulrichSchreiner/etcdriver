package main

import (
	"log"
	"os"
	"sync"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/xetorthio/etcd-fs/src/etcdfs"
)

var etcdbackend etcdfs.EtcdFs
var mountpoints map[string]*etcdFuseServer
var mux sync.Mutex

type etcdFuseServer struct {
	mountpoint string
	server     *fuse.Server
	count      int
}

func activate() []string {
	etcdbackend = etcdfs.EtcdFs{FileSystem: pathfs.NewDefaultFileSystem(), EtcdEndpoint: "http://localhost:4001"}
	mountpoints = make(map[string]*etcdFuseServer)
	return []string{"VolumeDriver"}
}

func create(volume string) error {
	log.Printf("create %s", volume)
	return nil
}

func remove(volume string) error {
	log.Printf("remove %s", volume)
	return nil
}

func mount(volume string) (string, error) {
	mux.Lock()
	defer mux.Unlock()

	s, ok := mountpoints[volume]
	if ok {
		s.count = s.count + 1
		return s.mountpoint, nil
	}

	log.Printf("mount volume: %s", volume)
	pt := "/tmp/" + volume
	if err := os.MkdirAll(pt, 0755); err != nil {
		return "", err
	}
	root := pathfs.NewPrefixFileSystem(&etcdbackend, "/"+volume)
	nfs := pathfs.NewPathNodeFs(root, nil)
	server, _, err := nodefs.MountRoot(pt, nfs.Root(), nil)
	if err != nil {
		return "", err
	}
	es := etcdFuseServer{pt, server, 0}
	mountpoints[volume] = &es

	go server.Serve()
	return pt, nil
}

func path(volume string) (string, error) {
	log.Printf("path %s", volume)
	return "/tmp/" + volume, nil
}

func unmount(volume string) error {
	mux.Lock()
	defer mux.Unlock()

	s, ok := mountpoints[volume]
	if ok {
		if s.count == 0 {
			delete(mountpoints, volume)
			return s.server.Unmount()
		} else {
			s.count = s.count - 1
		}
	}
	return nil
}

