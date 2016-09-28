package main

import (
	//"crypto/md5"
	//"encoding/hex"
	_ "bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime"
	"strings"
	//"sync"
	"time"

	"github.com/franela/goreq"
	"github.com/go-ini/ini"
	"github.com/nats-io/nats"
	"github.com/paulbellamy/ratecounter"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
	"github.com/yuin/gopher-lua"
	//"github.com/Shopify/go-lua"
	//"github.com/conformal/gotk3/gtk" // for http_browser tests
	//"github.com/sourcegraph/webloop" // for http_browser tests
)

type BrigadeNode struct {
	NodeID               string
	NumWorkers           int
	Config               *ini.File
	NatsConn             *nats.Conn
	Subscriptions        []*nats.Subscription
	StatsUpdatePeriod    int
	Debug                bool
	statsConn            *nats.Conn
	Stats                *NodeStats
	CompletedRequestList chan map[string]int
	RateCounter          *ratecounter.RateCounter
	RateCounterFailed    *ratecounter.RateCounter
	CommanderStatsUpdate time.Duration
	//PingTimeout          int
}

type NodeStats struct {
	PrevRequestsServed  int64
	TotalRequestsServed int64
}

const (
	PING_TIMEOUT = 30 // 90 seconds
)

var quit chan bool
var commanderping = make(chan string)
var node_id string

func init() {
	//node_id = GetRndStr(12)
	//fmt.Println("Setting node ID to", node_id)
}

func NewBrigadeNode(conf *ini.File, debug bool, updatePeriod int, numWorkers int) BrigadeNode {
	n := BrigadeNode{
		NodeID:               GetRndStr(12),
		Config:               conf,
		Debug:                true,
		NumWorkers:           numWorkers,
		StatsUpdatePeriod:    updatePeriod,
		CommanderStatsUpdate: (time.Duration(1) * time.Second),
	}

	quit = make(chan bool, numWorkers+1)
	n.RateCounter = ratecounter.NewRateCounter(1 * time.Second)
	n.RateCounterFailed = ratecounter.NewRateCounter(1 * time.Second)
	fmt.Printf("[INFO] Node started with ID %s\n", n.NodeID)
	return n
}

func (bn *BrigadeNode) SetId(id string) {
	bn.NodeID = id
}

func (bn *BrigadeNode) ConnectAndSubscribe() {

	opts := nats.DefaultOptions
	opts.Servers = strings.Split(bn.Config.Section("nats").Key("servers").String(), ",")
	for i, s := range opts.Servers {
		opts.Servers[i] = strings.Trim(s, " ")
	}

	/*
	   secure, err := conf.Section("nats").Key("enable_ssl").MustBool()
	   if err != nil {
	     opts.Secure = secure
	   }
	*/
	//secure := conf.Section("").Key("enable_ssl").MustBool(false)
	opts.Secure = false
	opts.MaxReconnect = 5
	opts.ReconnectWait = (2 * time.Second)

	//bn.StatsMutex = &sync.Mutex{}

	nc, err := opts.Connect()
	if err != nil {
		log.Fatalf("[ERROR] Can't connect: %v\n", err)
	}
	bn.NatsConn = nc

	bn.NatsConn.Opts.DisconnectedCB = func(_ *nats.Conn) {
		fmt.Printf("[INFO] Disconnected from NATS cluster!\n")
	}

	bn.NatsConn.Opts.ReconnectedCB = func(nc *nats.Conn) {
		fmt.Printf("[INFO] Reconnected to NATS cluster via %v!\n", nc.ConnectedUrl())
	}

	// Listen on the ping channel for check-ins from the commander
	bn.NatsConn.Subscribe("ping", func(msg *nats.Msg) {
		commanderping <- string(msg.Data)
	})

	// Initialize the RPS counter map
	bn.subscribe()
	bn.calculateAndSendRPS()
}

func (bn *BrigadeNode) subscribe() {

	if bn.Debug {
		fmt.Println("Subscribing to subjects:", directive_types)
	}
	for _, d := range directive_types {
		subs, _ := bn.NatsConn.Subscribe(d, func(msg *nats.Msg) {

			// Spawn the number of workers corresponding to the NumWorkers value
			for i := 0; i < bn.NumWorkers; i++ {
				go bn.ProcessDirective(msg, i)
			}
			if bn.Debug {
				fmt.Printf("[DEBUG] #%d workers are now processing directives.\n", bn.NumWorkers)
			}
		})
		bn.Subscriptions = append(bn.Subscriptions, subs)

	}

	bn.NatsConn.Subscribe("command", func(msg *nats.Msg) {
		bn.processCommandDirective(msg)
	})

	/*
	 In a seperate goroutine, start a time for received pings from the commander.
	 If no ping is received within a given delay, stop the worker
	*/
	go func(bn *BrigadeNode) {
		timer := time.NewTimer(time.Duration(bn.Config.Section("").Key("timeout").MustInt(PING_TIMEOUT)) * time.Second)
		for {
			select {
			case <-commanderping:
				timer.Reset(time.Duration(bn.Config.Section("").Key("timeout").MustInt(PING_TIMEOUT)) * time.Second)
			case <-timer.C:
				for i := 0; i < bn.NumWorkers; i++ {
					quit <- true
				}
			}
		}
	}(bn)

	// Listen for quit signal even if directives haven't been received
	go func() {
		for {
			select {
			case <-quit:
				fmt.Println("[NOTICE] Got halt directive.  Stoping worker.")
				return
			}
		}
	}()

}

func (bn *BrigadeNode) ProcessDirective(msg *nats.Msg, worker_num int) {

	if bn.Debug {
		//fmt.Printf("[NOTICE] Processing directive for subject %s\n", msg.Subject)
	}

	switch msg.Subject {
	case "http":
		bn.processHttpDirective(msg, worker_num)
	case "http_browser":
		bn.processHttpBrowserDirective(msg)
	case "ssh":
		bn.processSshDirective(msg)
	case "script":
		bn.processScriptDirective(msg, worker_num)
	default:
		fmt.Println("[ERROR] Unknow directive type:", msg.Subject)
	}

}

func (bn *BrigadeNode) processCommandDirective(msg *nats.Msg) {

	var d map[string]interface{}
	if err := json.Unmarshal(msg.Data, &d); err != nil {
		fmt.Println("[ERROR] Command Directive has invalid JSON")
		return
	}
	if bn.Debug {
		fmt.Println("[DEBUG] Command Directives:", d)
	}

	switch d["type"] {

	case "halt":
		fmt.Println("[NOTICE] Processing halt command directive.  Unsubscribed from all subjects once currect directive completed.")
		/*
			for _, subs := range bn.Subscriptions {
				subs.Unsubscribe()
			}
			bn.Subscriptions = bn.Subscriptions[:0]
		*/
		// Send a quit signal to each worker that was spawned
		for i := 0; i < bn.NumWorkers+1; i++ {
			fmt.Println("[DEBUG] Publishing quit singal for worker #", i)
			quit <- true
		}

	case "resume":
		fmt.Println("[NOTICE] Processing resume command directive.  Resubscribing to all subjects")
		for _, d := range directive_types {
			subs, _ := bn.NatsConn.Subscribe(d, func(msg *nats.Msg) {
				for i := 0; i < runtime.NumCPU(); i++ {
					if bn.Debug {
						fmt.Printf("\n[DEBUG] Node worker #%d processing directives.\n", i)
					}
					go bn.ProcessDirective(msg, i)
				}
			})
			bn.Subscriptions = append(bn.Subscriptions, subs)
		}
	case "shutdown":
		bn.Terminate()
	default:
		fmt.Println("[ERROR] Unknow command directive:", msg.Subject)

	}
}

func (bn *BrigadeNode) processHttpBrowserDirective(msg *nats.Msg) {
	var d HttpDirectives
	if err := json.Unmarshal(msg.Data, &d); err != nil {
		fmt.Println("[ERROR] HTTP Directive has invalid JSON:", err)
		//os.Exit(1)
		return
	}
	if bn.Debug {
		PrintDebug(3, "Processing http_browser directive - UNIMPLEMENTED")
	}

}

func (bn *BrigadeNode) processHttpDirective(msg *nats.Msg, workerNum int) {
	// Check if it's a valid JSON payload

	var directives_list HttpDirectives
	if err := json.Unmarshal(msg.Data, &directives_list); err != nil {
		fmt.Println("[ERROR] HTTP Directive has invalid JSON:", err)
		return
	}

	//if bn.Debug {
	//	fmt.Println("[DEBUG] HTTP Directive:", directives_list)
	//}

	directives := directives_list.Directives

	// Get total weight of all directives and build simple map that has the url and method as the key
	var total_weight int
	for _, dr := range directives {
		total_weight += dr.Weight
	}

	total_directives := len(directives)
	var dnum, nw, probability int

	for {

		if bn.Debug {
			fmt.Printf("[DEBUG] Worker #%d running request\n", workerNum)
		}

		dnum = rand.Intn(total_directives)
		nw = WeightToPercentage(directives[dnum].Weight, total_weight)
		probability = rand.Intn(100) + 1
		if probability >= nw {
			continue
		}

		d := directives[dnum]

		var req_start, req_end int64

		req := goreq.Request{
			Method:      strings.ToUpper(d.Method),
			Uri:         d.Url,
			Compression: goreq.Gzip(),
			UserAgent:   "Brigade Load Tester (" + VERSION + ")",
			Timeout:     1000 * time.Millisecond,
		}

		// If there's a proxy specified for the given url, then use it (TODO)
		//req.Proxy: "http://user:pass@myproxy:myproxyport"

		// If there are cookies, then include them (TODO)
		//req.WithCookie(&http.Cookie{Name: "c1", Value: "v1"})

		// If there is post data to send, add it (TODO)
		/*
			if len(d.PostData) >= 1 {
				req.Body = d.PostData
			}
		*/

		// If there are headers to specify, add them
		if len(d.Headers) >= 1 {
			for _, header := range d.Headers {
				req.AddHeader(header.Name, header.Value)
			}
		}

		req_start = time.Now().UnixNano() / int64(time.Millisecond)

		// Track request start
		res, err := req.Do()
		req_end = time.Now().UnixNano() / int64(time.Millisecond)

		// Buidl and send the stats object
		if err == nil {
			bn.RateCounter.Incr(1)

			body, _ := res.Body.ToString()
			bn.sendStats(&HttpDirectiveStats{
				Url:          d.Url,
				Method:       d.Method,
				RequestTime:  int64(req_end - req_start),
				ResponseSize: len([]byte(body)),
				ResponseCode: res.StatusCode,
			})

		} else {

			bn.RateCounterFailed.Incr(1)

			//body, err := res.Body.ToString()

			fmt.Printf("[ERROR] Failed request code: %s\n", err)

			bn.sendStats(&HttpDirectiveStats{
				Url:          d.Url,
				Method:       d.Method,
				RequestTime:  int64(req_end - req_start),
				ResponseSize: 0,
				ResponseCode: 0,
			})
		}

		if directives_list.MinDelay < 0 {
			directives_list.MinDelay = 0
		}
		if directives_list.MaxDelay < directives_list.MinDelay {
			directives_list.MaxDelay = 0
		}

		sleepTime := time.Duration(rand.Intn(directives_list.MaxDelay-directives_list.MinDelay)+directives_list.MinDelay) * time.Millisecond
		//if bn.Debug {
		//	fmt.Println("[DEBUG] Sleeping for ", sleepTime, "milliseconds")
		//}

		select {
		case <-quit:
			fmt.Printf("[NOTICE] Worker #%d received halt directive. Halting work.\n", workerNum)
			return
		default:

		}

		// Sleep for x milliseconds between MinDelay and MaxDelay until next request
		time.Sleep(sleepTime)

	}

}

func (bn *BrigadeNode) processScriptDirective(msg *nats.Msg, worker_num int) {

	var directives_list ScriptDirective
	if err := json.Unmarshal(msg.Data, &directives_list); err != nil {
		fmt.Println("[ERROR] Script Directive has invalid JSON:", err)
		//os.Exit(1)
		return
	}
	if bn.Debug {
		//fmt.Println("[DEBUG] HTTP Directives:", d)
	}

	L := lua.NewState()
	defer L.Close()
	/*
		if err := L.DoFile("hello.lua"); err != nil {
			fmt.Println()
		}
	*/
	if err := L.DoString(directives_list.ScriptBody); err != nil {
		fmt.Println("Could not execute lua code:", err)
	}
	ret := L.Get(-1)
	fmt.Println("Returned value:", ret)

}

func (bn *BrigadeNode) processSshDirective(msg *nats.Msg) {
	/*
		var directives_list SshDirectives
		if err := json.Unmarshal(msg.Data, &directives_list); err != nil {
			fmt.Println("[ERROR] SSH Directive has invalid JSON:", err)
			//os.Exit(1)
			return
		}
		if bn.Debug {
			PrintDebug(3, "Processing ssh directive - UNIMPLEMENTED")
		}

		ssh_session := NewSshClient("localhost", 22, "root", GetRndPass(8))

		directives := directives_list.Directives

		// Get total weight of all directives and build simple map that has the url and method as the key
		var total_weight int
		for _, d := range directives {
			total_weight += d.Weight
		}

		total_directives := len(directives)

		var dnum, nw, probability int
		var b bytes.Buffer

		for {

			dnum = rand.Intn(total_directives)
			nw = WeightToPercentage(directives[dnum].Weight, total_weight)
			probability = rand.Intn(100) + 1
			if probability >= nw {
				continue
			}

			directive := directives[dnum]

			ssh_session.Stdout = &b
			// Need to get the last exit code to register failure or success
			if err := ssh_session.Run(directive.Command); err != nil {
				panic("Failed to run: " + err.Error())
			}

			// This is the result string
			fmt.Println(b.String())

			bn.sendStats(&SshDirectiveStats{
				Host:        directive.Host + ":" + directive.Port,
				Credentials: directive.Username + ":" + directive.Password,
				ExecTime:    int64(req_end - req_start),
				Command:     len([]byte(body)),
				ExitCode:    exit_code,
			})

			if directives_list.MinDelay < 0 {
				directives_list.MinDelay = 0
			}
			if directives_list.MaxDelay < directives_list.MinDelay {
				directives_list.MaxDelay = 0
			}

			select {
			case <-quit:
				fmt.Println("[NOTICE] Got halt directive.  Stoping worker.")
				return
			case bn.RpsChannel <- 1:
			}

			// Sleep for x milliseconds between MinDelay and MaxDelay until next request
			time.Sleep(time.Duration(rand.Intn(directives_list.MaxDelay-directives_list.MinDelay)+directives_list.MinDelay) * time.Millisecond)

		}
	*/

}

func (bn *BrigadeNode) Terminate() {
	bn.NatsConn.Close()
	fmt.Println("[NOTICE] Process terminating.")
	os.Exit(0)
}

func (bn *BrigadeNode) sendStats(s *HttpDirectiveStats) {
	payload, _ := json.Marshal(s)
	bn.NatsConn.Publish("stats", payload)
}

func (bn *BrigadeNode) calculateAndSendRPS() {

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	for _ = range time.Tick(bn.CommanderStatsUpdate) {
		// Now send the value on stats/rps queue subject
		// Also, set an interval at which you'll get CPU/Mem stats of this node and report back to the commander
		l, _ := load.Avg()
		cpuPercent, _ := cpu.Percent((time.Duration(1) * time.Second), false)
		vmem, _ := mem.VirtualMemory()

		payload, _ := json.Marshal(map[string]interface{}{
			"hostname": hostname,
			"id":       bn.NodeID,
			/*
				"system": map[string]interface{}{
					"load": map[string]interface{}{
						"onemin":     l.Load1,
						"fivemin":    l.Load5,
						"fifteenmin": l.Load15,
					},
					"cpu": RoundUp(cpuPercent[0], 1),
					"memory": map[string]interface{}{
						"total":       vmem.Total,
						"free":        vmem.Free,
						"usedpercent": RoundUp(vmem.UsedPercent, 1),
					},
				},
			*/
			"cpupercent":     RoundUp(cpuPercent[0], 1),
			"load1min":       l.Load1,
			"load5min":       l.Load5,
			"load15min":      l.Load15,
			"memtotal":       vmem.Total,
			"memfree":        vmem.Free,
			"memusedpercent": RoundUp(vmem.UsedPercent, 1),
			"rps":            bn.RateCounter.Rate(),
			"rpsfailed":      bn.RateCounterFailed.Rate(),
		})
		fmt.Printf("[DEBUG] Sending RPS: %s\n", payload)
		bn.NatsConn.Publish("progress", payload)
		runtime.Gosched()
	}
}
