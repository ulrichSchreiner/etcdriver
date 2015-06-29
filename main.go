package main

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gocraft/web"
)

type volumeDriver struct {
}

type createRq struct {
	Name string
}
type createRsp struct {
	Err string
}

type removeRq struct {
	Name string
}
type removeRsp struct {
	Err string
}

type mountRq struct {
	Name string
}
type mountRsp struct {
	Err        string
	Mountpoint string
}

type unmountRq struct {
	Name string
}
type unmountRsp struct {
	Err string
}

type pathRq struct {
	Name string
}
type pathRsp struct {
	Err        string
	Mountpoint string
}

type activateRsp struct {
	Implements []string
}

func (c *volumeDriver) activatePlugin(rw web.ResponseWriter, req *web.Request) {
	json.NewEncoder(rw).Encode(activateRsp{activate()})
}

func (c *volumeDriver) createVolume(rw web.ResponseWriter, req *web.Request) {
	var rq createRq
	var rsp createRsp
	if err := json.NewDecoder(req.Body).Decode(&rq); err != nil {
		rsp.Err = err.Error()
	} else {
		err := create(rq.Name)
		if err != nil {
			rsp.Err = err.Error()
		}
	}
	json.NewEncoder(rw).Encode(rsp)
}

func (c *volumeDriver) removeVolume(rw web.ResponseWriter, req *web.Request) {
	var rq removeRq
	var rsp removeRsp
	if err := json.NewDecoder(req.Body).Decode(&rq); err != nil {
		rsp.Err = err.Error()
	} else {
		err := remove(rq.Name)
		if err != nil {
			rsp.Err = err.Error()
		}
	}
	json.NewEncoder(rw).Encode(rsp)
}

func (c *volumeDriver) mountVolume(rw web.ResponseWriter, req *web.Request) {
	var rq mountRq
	var rsp mountRsp
	if err := json.NewDecoder(req.Body).Decode(&rq); err != nil {
		rsp.Err = err.Error()
	} else {
		pt, err := mount(rq.Name)
		if err != nil {
			rsp.Err = err.Error()
		} else {
			rsp.Mountpoint = pt
		}
	}
	json.NewEncoder(rw).Encode(rsp)
}

func (c *volumeDriver) unmountVolume(rw web.ResponseWriter, req *web.Request) {
	var rq unmountRq
	var rsp unmountRsp
	if err := json.NewDecoder(req.Body).Decode(&rq); err != nil {
		rsp.Err = err.Error()
	} else {
		err := unmount(rq.Name)
		if err != nil {
			rsp.Err = err.Error()
		}
	}
	json.NewEncoder(rw).Encode(rsp)
}

func (c *volumeDriver) pathVolume(rw web.ResponseWriter, req *web.Request) {
	var rq pathRq
	var rsp pathRsp
	if err := json.NewDecoder(req.Body).Decode(&rq); err != nil {
		rsp.Err = err.Error()
	} else {
		pt, err := path(rq.Name)
		if err != nil {
			rsp.Err = err.Error()
		} else {
			rsp.Mountpoint = pt
		}
	}
	json.NewEncoder(rw).Encode(rsp)
}

func main() {
	router := web.New(volumeDriver{}).
		Middleware(web.LoggerMiddleware).
		Post("/Plugin.Activate/", (*volumeDriver).activatePlugin).
		Post("/VolumeDriver.Create/", (*volumeDriver).createVolume).
		Post("/VolumeDriver.Remove/", (*volumeDriver).removeVolume).
		Post("/VolumeDriver.Mount/", (*volumeDriver).mountVolume).
		Post("/VolumeDriver.Unmount/", (*volumeDriver).unmountVolume).
		Post("/VolumeDriver.Path/", (*volumeDriver).pathVolume)

	l, err := net.Listen("unix", "/usr/share/docker/plugins/etcdriver.sock")
	if err != nil {
		log.Fatal("listen error:", err)
	}
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill, syscall.SIGTERM)
	go func(c chan os.Signal) {
		sig := <-c
		log.Printf("Caught signal %s: shutting down.", sig)
		l.Close()
		os.Exit(0)
	}(sigc)

	log.Fatal(http.Serve(l, router))
}
