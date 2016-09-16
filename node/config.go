package main

import (
	"fmt"
	"github.com/go-ini/ini"
	"os"
	//"runtime"
)

type BrigadeConfig struct {
	Servers           []string `json:"servers"`
	NatsMaxReconnect  int      `json:"nats_max_reconect"`
	NatsReconnectWait int      `json:"nats_reconect_wait"`
}

//var nats_conn *nats.Conn
//var nats_encoded_conn *nats.EncodedConn
//var conf *ini.File

//var num_workers = runtime.NumCPU() // Should eventually be moved the the .ini config

//var conf *BrigadeConfig
var directive_types = []string{"http", "ssh", "exec", "script"}

const (
	DEBUG = true
)

func LoadConfig(f string) (conf *ini.File) {

	if DEBUG {
		fmt.Println("[DEBUG] Loading config file: ", f)
	}
	conf, err := ini.Load(f)

	if err != nil {
		fmt.Println("[ERROR] Could not load config: ", err)
		os.Exit(1)
	}
	return conf

}
