package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/calavera/dkvolume"
	"github.com/hanwen/go-fuse/fuse"
	"github.com/hanwen/go-fuse/fuse/nodefs"
	"github.com/hanwen/go-fuse/fuse/pathfs"
	"github.com/xetorthio/etcd-fs/src/etcdfs"
)

type etcdFuseServer struct {
	mountpoint string
	server     *fuse.Server
	count      int
}

type etcDriver struct {
	etcdbackend etcdfs.EtcdFs
	mountpoints map[string]*etcdFuseServer
	mux         sync.Mutex
	basePath    string
}

func responseError(e error) dkvolume.Response {
	if e != nil {
		return responseErrorString(e.Error())
	}
	return dkvolume.Response{}
}

func responseErrorString(e string) dkvolume.Response {
	return dkvolume.Response{
		Err: e,
	}
}

func NewDriver(basepath, etcdendpoint string) dkvolume.Driver {
	e := etcDriver{
		etcdbackend: etcdfs.EtcdFs{FileSystem: pathfs.NewDefaultFileSystem(), EtcdEndpoint: etcdendpoint},
		mountpoints: make(map[string]*etcdFuseServer),
		basePath:    basepath,
	}
	return &e
}

func (d *etcDriver) createPath(p string) string {
	return fmt.Sprintf("%s/%d", d.basePath, time.Now().Nanosecond())
}

func (d *etcDriver) asEtcdRoot(r string) string {
	return string([]byte(r)[1:])
}

func (d *etcDriver) Create(rq dkvolume.Request) dkvolume.Response {
	var res dkvolume.Response
	if !strings.HasPrefix(rq.Name, "@") {
		return responseErrorString("a etcd path has to start with a @")
	}
	d.mux.Lock()
	defer d.mux.Unlock()

	_, ok := d.mountpoints[rq.Name]
	if ok {
		return res
	}

	pt := d.createPath(rq.Name)
	if err := os.MkdirAll(pt, 0755); err != nil {
		return responseError(fmt.Errorf("cannot mkdir: %s", err))
	}
	rootfs := "/" + d.asEtcdRoot(rq.Name)
	root := pathfs.NewPrefixFileSystem(&d.etcdbackend, rootfs)
	nfs := pathfs.NewPathNodeFs(root, nil)
	server, _, err := nodefs.MountRoot(pt, nfs.Root(), nil)

	if err != nil {
		return responseError(fmt.Errorf("cannot mount root : %s", err))
	}
	es := etcdFuseServer{pt, server, 0}
	d.mountpoints[rq.Name] = &es

	go server.Serve()
	return res
}

func (d *etcDriver) Remove(rq dkvolume.Request) dkvolume.Response {
	d.mux.Lock()
	defer d.mux.Unlock()

	s, ok := d.mountpoints[rq.Name]
	if ok {
		if s.count < 1 {
			delete(d.mountpoints, rq.Name)
			if e := s.server.Unmount(); e != nil {
				return responseError(e)
			}
			return responseError(os.Remove(s.mountpoint))
		}
	}
	return dkvolume.Response{}
}

func (d *etcDriver) Path(rq dkvolume.Request) dkvolume.Response {
	d.mux.Lock()
	defer d.mux.Unlock()

	s, ok := d.mountpoints[rq.Name]
	if ok {
		return dkvolume.Response{Mountpoint: s.mountpoint}
	}

	return responseError(fmt.Errorf("%s not found", rq.Name))
}

func (d *etcDriver) Mount(rq dkvolume.Request) dkvolume.Response {
	d.mux.Lock()
	defer d.mux.Unlock()

	s, ok := d.mountpoints[rq.Name]
	if ok {
		s.count++
		return dkvolume.Response{Mountpoint: s.mountpoint}
	}

	return responseError(fmt.Errorf("%s not created", rq.Name))
}

func (d *etcDriver) Unmount(rq dkvolume.Request) dkvolume.Response {
	d.mux.Lock()
	defer d.mux.Unlock()

	s, ok := d.mountpoints[rq.Name]
	if ok {
		s.count++
	}
	return dkvolume.Response{}
}
