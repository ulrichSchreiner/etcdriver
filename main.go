package main

import (
	"fmt"

	"github.com/calavera/dkvolume"
	"github.com/spf13/viper"
)

const (
	basepathKey   = "BASEPATH"
	etcdurlKey    = "ETCDURL"
	socketAddress = "/usr/share/docker/plugins/etcdriver.sock"
)

func main() {
	viper.SetEnvPrefix("ETCDRIVER")
	viper.AutomaticEnv()
	viper.SetDefault(basepathKey, "/tmp/etcdriver")
	viper.SetDefault(etcdurlKey, "http://localhost:4001")
	base := viper.GetString(basepathKey)
	etcdurl := viper.GetString(etcdurlKey)

	driver := NewDriver(base, etcdurl)
	h := dkvolume.NewHandler(driver)
	fmt.Printf("Listening on %s\n", socketAddress)
	fmt.Println(h.ServeUnix("root", socketAddress))
}
