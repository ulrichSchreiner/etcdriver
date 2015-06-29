package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/xetorthio/etcd-fs/src/etcdfs"
)

var etcdbackend etcdfs.EtcdFs
var mountpoints map[string]*etcdFuseServer
var mux sync.Mutex

const basePath = "/tmp/etcd"

type etcdFuseServer struct {
	mountpoint string
	server     *fuse.Server
	count      int
}

func createPath(p string) string {
	return filepath.Join(basePath, asEtcdRoot(p))
}

func asEtcdRoot(r string) string {
	return string([]byte(r)[1:])
}

func activate() []string {
	etcdbackend = etcdfs.EtcdFs{FileSystem: pathfs.NewDefaultFileSystem(), EtcdEndpoint: "http://localhost:4001"}
	mountpoints = make(map[string]*etcdFuseServer)
	return []string{"VolumeDriver"}
}

func create(volume string) error {
	log.Printf("create %s", volume)
	if !strings.HasPrefix(volume, "@") {
		return fmt.Errorf("a etcd path has to start with a @")
	}
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

	pt := createPath(volume)
	if err := os.MkdirAll(pt, 0755); err != nil && !os.IsExist(err) {
		return "", fmt.Errorf("cannot mk dir: %s", err)
	}
	rootfs := "/" + asEtcdRoot(volume)
	root := pathfs.NewPrefixFileSystem(&etcdbackend, rootfs)
	nfs := pathfs.NewPathNodeFs(root, nil)
	server, _, err := nodefs.MountRoot(pt, nfs.Root(), nil)

	//nfs := pathfs.NewPathNodeFs(&etcdbackend, nil)
	//log.Printf("mounting to: %s", pt)
	//server, _, err := nodefs.MountRoot(pt, nfs.Root(), nil)
	if err != nil {
		return "", fmt.Errorf("cannot mount root : %s", err)
	}
	es := etcdFuseServer{pt, server, 0}
	mountpoints[volume] = &es

	go server.Serve()
	return pt, nil
}

func path(volume string) (string, error) {
	return createPath(volume), nil
}

func unmount(volume string) error {
	mux.Lock()
	defer mux.Unlock()

	v := asEtcdRoot(volume)
	s, ok := mountpoints[v]
	if ok {
		if s.count == 0 {
			delete(mountpoints, v)
			return s.server.Unmount()
		} else {
			s.count = s.count - 1
		}
	}
	return nil
}
