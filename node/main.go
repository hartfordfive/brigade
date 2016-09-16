package main

import (
	"flag"
	"log"
	"runtime"
	//"strings"
	//"sync"
	"fmt"
	"os"
	"os/signal"
	//"time"
	"github.com/go-ini/ini"
)

func usage() {
	log.Fatalf("Usage: brigade [-c config]\n")
}

func main() {

	//var debug = flag.Bool("d", false, "Debug mode")
	var conf_file = flag.String("c", "node.ini", "The configuration file")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	//args := flag.Args()

	var conf *ini.File
	if _, err := os.Stat(*conf_file); err == nil {
		conf = LoadConfig(*conf_file)
	} else {
		fmt.Println("Error: Config file ", conf_file, "does not exist!")
		os.Exit(1)
	}

	num_workers := conf.Section("").Key("num_workers").MustInt(runtime.NumCPU())
	updatePeriod := conf.Section("").Key("update_period").MustInt(5)
	/*
		bn := &BrigadeNode{
			Config: conf, Debug: true,
			NumWorkers: num_workers}
	*/

	bn := NewBrigadeNode(conf, true, updatePeriod, num_workers)

	// Connect and subscribe to the directive message queues
	bn.ConnectAndSubscribe()

	log.SetFlags(log.LstdFlags)

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		fmt.Println("[NOTICE] Caught interrupt signal.")
		fmt.Println("[NOTICE] Closing connections to NATS server.")
		bn.Terminate()
	}()

	runtime.Goexit()
}
