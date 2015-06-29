package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/xetorthio/etcd-fs/src/etcdfs"
)

var (
	etcdbackend  etcdfs.EtcdFs
	mountpoints  map[string]*etcdFuseServer
	mux          sync.Mutex
	basePath     string
	etcdEndpoint string
)

func init() {
	basePath = os.Getenv("ETCDRIVER_BASE")
	if basePath == "" {
		basePath = "/tmp/etcd"
	}
	etcdEndpoint = os.Getenv("ETCD_ENDPOINT")
	if etcdEndpoint == "" {
		etcdEndpoint = "http://localhost:4001"
	}
}

type etcdFuseServer struct {
	mountpoint string
	server     *fuse.Server
	count      int
}

func createPath(p string) string {
	return fmt.Sprintf("%s/%d", basePath, time.Now().Nanosecond())
}

func asEtcdRoot(r string) string {
	return string([]byte(r)[1:])
}

func activate() []string {
	etcdbackend = etcdfs.EtcdFs{FileSystem: pathfs.NewDefaultFileSystem(), EtcdEndpoint: etcdEndpoint}
	mountpoints = make(map[string]*etcdFuseServer)
	return []string{"VolumeDriver"}
}

func create(volume string) error {
	if !strings.HasPrefix(volume, "@") {
		return fmt.Errorf("a etcd path has to start with a @")
	}
	mux.Lock()
	defer mux.Unlock()

	_, ok := mountpoints[volume]
	if ok {
		return nil
	}

	pt := createPath(volume)
	if err := os.MkdirAll(pt, 0755); err != nil {
		return fmt.Errorf("cannot mkdir: %s", err)
	}
	rootfs := "/" + asEtcdRoot(volume)
	root := pathfs.NewPrefixFileSystem(&etcdbackend, rootfs)
	nfs := pathfs.NewPathNodeFs(root, nil)
	server, _, err := nodefs.MountRoot(pt, nfs.Root(), nil)

	if err != nil {
		return fmt.Errorf("cannot mount root : %s", err)
	}
	es := etcdFuseServer{pt, server, 0}
	mountpoints[volume] = &es

	go server.Serve()
	return nil
}

func remove(volume string) error {
	mux.Lock()
	defer mux.Unlock()

	s, ok := mountpoints[volume]
	if ok {
		if s.count == 0 {
			delete(mountpoints, volume)
			if e := s.server.Unmount(); e != nil {
				return e
			}
			return os.Remove(s.mountpoint)
		}
	}
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

	return "", fmt.Errorf("%s not created", volume)
}

func path(volume string) (string, error) {
	mux.Lock()
	defer mux.Unlock()

	s, ok := mountpoints[volume]
	if ok {
		return s.mountpoint, nil
	}

	return "", fmt.Errorf("%s not found", volume)
}

func unmount(volume string) error {
	mux.Lock()
	defer mux.Unlock()

	s, ok := mountpoints[volume]
	if ok {
		s.count = s.count - 1
	}
	return nil
}
