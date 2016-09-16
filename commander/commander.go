package main

import (
	//"bytes"
	//"crypto/sha1"
	"encoding/json"
	//"errors"
	//"flag"
	"fmt"
	//"io"
	//"io/ioutil"
	//"log"
	//"mime/multipart"
	//"net/http"
	"os"
	"os/signal"
	//"runtime"
	//"strings"
	"time"

	//"github.com/facebookgo/grace/gracehttp"
	"github.com/go-ini/ini"
	"github.com/nats-io/nats"
	//"github.com/paulbellamy/ratecounter"
)

type CommanderNode struct {
	Config        *ini.File
	Subscriptions []*nats.Subscription
	Debug         bool
	NatsConn      *nats.Conn
	Stats         *map[string]interface{}
	//PingTimeout          int
	ClusterHttpDirectiveStats
	DirectivesInProgress bool
	DirectivesList       map[string]interface{}
	NodeList             map[string]map[string]interface{}
}

const (
//DEBUG      = true
//PING_DELAY = 15 // 15 seconds
)

func NewCommanderNode() *CommanderNode {

	cn := &CommanderNode{
		NodeList: map[string]map[string]interface{}{},
	}
	return cn
}

// Load the INI config file
func (cn *CommanderNode) LoadConfig(config_file string) *ini.File {
	if DEBUG {
		fmt.Println("[DEBUG] Loading config file: ", config_file)
	}
	conf, err := ini.Load(config_file)

	if err != nil {
		fmt.Println("[ERROR] Could not load config: ", err)
		os.Exit(1)
	}
	return conf
}

// Connect to the relevant NATS messaging queues
func (cn *CommanderNode) ConnectAndSubscribe() {

	cn.NatsConn.QueueSubscribe("stats", "stats", func(msg *nats.Msg) {
		cn.ProcessHttpStats(msg)
	})
	cn.NatsConn.QueueSubscribe("progress", "rps", func(msg *nats.Msg) {
		cn.ProcessRps(msg.Data)
	})
	cn.NatsConn.Subscribe("queries", func(msg *nats.Msg) {
		// TO COMPLETE

		/*
		   Respond to the individual queries from the nodes here.
		   For example:,
		   1) A node may send a query to have the directives re-sent to it if it's
		   joining a group of nodes that's already running directives.  Will have to
		   determine a good mechanism for this.
		*/

	})

	go cn.checkInWithNodes(nc, time.Duration(cn.Config.Section("").Key("ping_delay").MustInt(PING_DELAY)))

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		fmt.Println("[NOTICE] Caught interrupt signal, terminated commander.")
		os.Exit(0)
	}()

}

func (cn *CommanderNode) ProcessHttpStats(msg *nats.Msg) {

	var d HttpDirectiveStats
	if err := json.Unmarshal(msg.Data, &d); err != nil {
		fmt.Println("[ERROR] Stats payload has invalid JSON")
		return
	}

	cs.TotalBytesDownloaded += d.ResponseSize
	cs.TotalRequestsCompleted += 1
	cs.UrlHits[d.Method+"~"+d.Url] += 1

	if d.ResponseCode == -1 {
		cs.TotalRequestsFailed += 1
	}

	cs.UrlRequestsCompleted[d.Method+"~"+d.Url] = map[string]uint64{
		"total_hits":         cs.UrlHits[d.Method+"~"+d.Url],
		"last_response_code": uint64(d.ResponseCode),
		"last_response_size": d.ResponseSize,
		"last_request_time":  uint64(d.RequestTime),
	}

}

func (cn *CommanderNode) ProcessRps(m []byte) {

	var d map[string]interface{}
	if err := json.Unmarshal(m, &d); err != nil {
		fmt.Println("[ERROR] Unknow format for RPS data")
		return
	}

	//val, ok := d["rps"].(float64)
	val, ok := d["rps"].(float64)
	if !ok && DEBUG {
		fmt.Println("[ERROR] Unknow rps value")
	}
	//cs.GlobalRps = int64(val)
	if ok {
		cs.ClusterRPS.Incr(int64(val))
	}

	val, ok = d["rps_failed"].(float64)
	if !ok && DEBUG {
		fmt.Println("[ERROR] Unknow rps value")
	}
	if ok {
		cs.ClusterFailedRPS.Incr(int64(val))
	}

	hostname, ok := d["hostname"].(string)
	id, ok := d["id"].(string)

	// Initialize that map index if not already sets
	if _, ok := node_list[id]; !ok {
		node_list[id] = map[string]interface{}{}
		node_list[id]["last_checkin"] = time.Now().UnixNano()
	}

	// Set the status as OK if checked in within the last minute
	last_checkin := node_list[id]["last_checkin"].(int64)
	curr_time := time.Now().UnixNano() / int64(time.Millisecond)
	if curr_time-last_checkin > 60000 {
		node_list[id]["status"] = 0
	} else {
		node_list[id]["status"] = 1
	}

	node_list[id]["last_checkin"] = time.Now().UnixNano() / int64(time.Millisecond)
	node_list[id]["hostname"] = hostname
}

// Check-in with all nodes bing sendinga ping every X seconds
func (cn *CommanderNode) checkInWithNodes(nc *nats.Conn, delay time.Duration) {
	for _ = range time.Tick(delay * time.Second) {
		nc.Publish("ping", []byte("checkin"))
	}
}
