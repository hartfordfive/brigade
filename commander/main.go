package main

import (
	//"bytes"
	//"crypto/sha1"
	"encoding/json"
	//"errors"
	"flag"
	"fmt"
	//"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/facebookgo/grace/gracehttp"
	"github.com/go-ini/ini"
	"github.com/nats-io/nats"
	"github.com/paulbellamy/ratecounter"
)

const (
	DEBUG      = true
	PING_DELAY = 15 // 15 seconds
)

//var ssl bool
var conf_file *string
var conf *ini.File
var last_rps_count, curr_rps_count int64
var cs *ClusterHttpDirectiveStats
var task_in_progress bool

//var ns *NodeStats
var nc *nats.Conn

//var node_list map[string]map[string]interface{}
var node_list map[string]interface{}
var directives_list map[string]interface{}

func usage() {
	log.Fatalf("Usage: commander [-c conf]\n")
}

var debug bool

func main() {

	//ssl = flag.Bool("ssl", false, "Use Secure Connection")
	conf_file = flag.String("c", "commander.ini", "The configuration file")

	log.SetFlags(0)
	flag.Usage = usage
	flag.Parse()

	//args := flag.Args()

	// If config was passed, get options that way, otherwise get servers from cli input
	opts := nats.DefaultOptions
	if _, err := os.Stat(*conf_file); err == nil {
		conf = loadConfig(*conf_file)
		opts.Servers = strings.Split(conf.Section("nats").Key("servers").String(), ",")
		for i, s := range opts.Servers {
			opts.Servers[i] = strings.Trim(s, " ")
		}

	} else {
		fmt.Println("[Error] Config file ", conf_file, "does not exist!")
		os.Exit(1)
	}

	opts.Secure = false
	opts.MaxReconnect = 5
	opts.ReconnectWait = (2 * time.Second)

	nc, err := opts.Connect()
	if err != nil {
		log.Fatalf("[ERROR] Can't connect: %v\n", err)
	}

	debug = conf.Section("").Key("debug").MustBool(false)

	/*
			   data, err :=
		     (conf.Section("directives").Key("file").String())
			   if err != nil {
			     fmt.Println("[Error] Could not get data from directives file: ", err)
			     os.Exit(1)
			   }
	*/

	// Will be removed once the directives can be uploaded via the web interface
	/*
	   nc.Publish(
	     conf.Section("directives").Key("type").String(),
	     []byte(data),
	   )
	*/
	//fmt.Printf("[DEBUG] Published '%s' type directives.\n", conf.Section("directives").Key("type").String())

	//log.Printf("Dircectves have been sent to the '%s' subject.\n", conf.Section("directives").Key("type").String())

	// Subscribe to the status subject to recieve and agregate all node stats
	nc.QueueSubscribe("stats", "stats", func(msg *nats.Msg) {
		ProcessStats(msg)
	})
	nc.QueueSubscribe("progress", "rps", func(msg *nats.Msg) {
		ProcessRps(msg.Data)
	})

	nc.Subscribe("queries", func(msg *nats.Msg) {
		// TO COMPLETE

		/*
		   Respond to the individual queries from the nodes here.
		   For example:,
		   1) A node may send a query to have the directives re-sent to it if it's
		   joining a group of nodes that's already running directives.  Will have to
		   determine a good mechanism for this.
		*/
		//if err := json.Unmarshal(msg.Data, &q); err != nil {
		//ProcessQuery(msg.Data, nc)

	})

	go checkInWithNodes(nc, time.Duration(conf.Section("").Key("ping_delay").MustInt(PING_DELAY)))

	log.SetFlags(log.LstdFlags)

	cs = &ClusterHttpDirectiveStats{
		UrlRequestsCompleted: map[string]map[string]uint64{},
		UrlHits:              map[string]uint64{},
		ClusterRPS:           ratecounter.NewRateCounter(1 * time.Second),
		ClusterFailedRPS:     ratecounter.NewRateCounter(1 * time.Second),
	}

	//node_list = map[string]map[string]interface{}{}
	node_list = map[string]interface{}{}

	task_in_progress = false

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		fmt.Println("[NOTICE] Caught interrupt signal, terminated commander.")
		os.Exit(0)
	}()

	gracehttp.Serve(
		&http.Server{Addr: "0.0.0.0:8082", Handler: httpHandler(nc)},
	)

	runtime.Goexit()
}

func ProcessStats(msg *nats.Msg) {

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

func ProcessRps(m []byte) {

	var d map[string]interface{}
	if err := json.Unmarshal(m, &d); err != nil {
		fmt.Println("[ERROR] Unknow format for RPS data")
		return
	}

	if DEBUG {
		fmt.Printf("[DEBUG] RPS Stats Payload: %v\n", &d)
	}

	// Add up RPS from each server and worker
	val, ok := d["rps"].(float64)
	if !ok && DEBUG {
		fmt.Println("[ERROR] Unknow rps value")
	}
	cs.ClusterRPS.Incr(int64(val))

	val, ok = d["rps_failed"].(float64)
	if !ok && DEBUG {
		fmt.Println("[ERROR] Unknow rps value")
	}
	cs.ClusterRPS.Incr(int64(val))

	hostname, ok := d["hostname"].(string)
	id, ok := d["id"].(string)

	// Initialize that map index if not already sets
	if _, ok := node_list[id]; !ok {
		node_list[id] = map[string]interface{}{}
		node_list[id]["last_checkin"] = time.Now().UnixNano()
	}

	// Set the status as OK if checked in within the last 3 minutes
	last_checkin := node_list[id]["last_checkin"].(int64)
	curr_time := time.Now().UnixNano() / int64(time.Millisecond)
	if curr_time-last_checkin > 180000 {
		node_list[id]["status"] = 0
	} else {
		node_list[id]["status"] = 1
	}

	node_list[id]["last_checkin"] = time.Now().UnixNano() / int64(time.Millisecond)
	node_list[id]["hostname"] = hostname
	node_list[id]["system"] = map[string]interface{}{}
}

/*
func ProcessQuery(msg string, nc *nats.Conn) {

  var q map[string]interface{}
  err := json.Unmarshal(msg, &q)
  if err != nil {
    fmt.Println("Error: Could not parse query JSON")
  }

  qtype, ok := q["type"].(string)
  if qtype == "resend_directives" {
    nc.Publish(directive_type, directives_list)
  }
}
*/

func loadConfig(f string) (conf *ini.File) { // CONVERTED

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

/*  FUNCTION NO LONGER USED - VERIFY THIS
func loadDirectivesFile(filename string) (string, error) {
	buf := bytes.NewBuffer(nil)
	f, err := os.Open(filename) // Error handling elided for brevity.
	if err != nil {
		return "", errors.New("[ERROR] Directives file does not exist!")
	}

	io.Copy(buf, f) // Error handling elided for brevity.
	f.Close()
	return string(buf.Bytes()), nil
}
*/

func httpHandler(nc *nats.Conn) http.Handler {

	mux := http.NewServeMux()

	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {

		// Set the current cluster-wide RPS
		cs.TotalRPS = cs.ClusterRPS.Rate()
		cs.TotalFailedRPS = cs.ClusterFailedRPS.Rate()
		data, err := json.Marshal(cs)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-control", "public, max-age=0")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "Brigade/"+VERSION)

		if err == nil {
			fmt.Fprintf(w, string(data))
		} else {
			fmt.Fprintf(w, "{\"status\": \"ERROR\"}")
		}

	})

	mux.HandleFunc("/api/current_nodes", func(w http.ResponseWriter, r *http.Request) {

		data, err := json.Marshal(node_list)

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-control", "public, max-age=0")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "Brigade/"+VERSION)

		if err == nil {
			fmt.Fprintf(w, string(data))
		} else {
			fmt.Fprintf(w, "{\"status\": \"ERROR\"}")
		}

	})

	mux.HandleFunc("/api/commands", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-control", "public, max-age=0")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "Brigade/"+VERSION)

		// payload should be: {"command": "halt"}
		//nc.Publish("command", payload)

		fmt.Fprintf(w, "{\"status\": \"NOT_IMPLEMENTED\"}")
	})

	mux.HandleFunc("/api/halt", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-control", "public, max-age=0")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "Brigade/"+VERSION)

		nc.Publish("command", []byte("{\"type\": \"halt\"}"))

		fmt.Fprintf(w, "{\"status\": \"ok\", \"message\": \"halt command sent\"}")
	})

	mux.HandleFunc("/api/shutdown", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-control", "public, max-age=0")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "Brigade/"+VERSION)

		nc.Publish("command", []byte("{\"type\": \"shutdown\"}"))

		fmt.Fprintf(w, "{\"status\": \"ok\", \"message\": \"shutdown command sent\"}")
	})

	mux.HandleFunc("/api/update_directives", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-control", "public, max-age=0")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "Brigade/"+VERSION)

		/*
		   NOTE: Need to ensure that if a list of directives is already running, posting a new set of directives
		         will either:
		           1. Fail and return warning that directives list already in progress
		           2. Notify use that the current directives will be stopped and new ones will start
		*/

		var (
			status int
			err    error
		)
		const _24K = (1 << 20) * 24
		if err = r.ParseMultipartForm(_24K); nil != err {
			fmt.Println("Error:", err)
			fmt.Fprintf(w, "{\"status\": \"error\", \"message\": \""+err.Error()+"\"}")
			status = http.StatusInternalServerError
			return
		}
		for _, fheaders := range r.MultipartForm.File {
			for _, hdr := range fheaders {
				// open uploaded
				var infile multipart.File
				if infile, err = hdr.Open(); nil != err {
					status = http.StatusInternalServerError
					fmt.Println("DEBUG:", status)
					fmt.Fprintf(w, "{\"status\": \"error\"}")
					return
				}

				filedata, err := ioutil.ReadAll(infile)
				if err != nil {
					fmt.Fprintf(w, "{\"status\": \"error\"}")
					return
				}

				// Ensure that it follows valid schema
				filedata_str := string(filedata)
				is_valid_http := false
				is_valid_script := false
				var dlist interface{}

				dlist, err = ValidateHttpDirective(filedata_str)
				if err == nil {
					is_valid_http = true
				} else {
					dlist, err = ValidateScriptDirective(filedata_str)
					if err == nil {
						is_valid_script = true
					}
				}

				directives_json, _ := json.Marshal(dlist)
				err = json.Unmarshal(directives_json, &directives_list)

				if !is_valid_http && !is_valid_script {
					fmt.Fprintf(w, "{\"status\": \"error\", \"message\": \"invalid directive format\"}")
					return
				}

				// Persist directives file to disk if enabled
				dtype, ok := directives_list["type"].(string)
				if ok {
					success, err := PersistDirectivesToDisk(directives_list, dtype+"-directives-"+YmdAndTimeToString()+".state")
					if !success {
						fmt.Println("[ERROR] ", err)
					}
				}

				// parse the json data
				//var decoded map[string]interface{}
				/*
				   if err := json.Unmarshal(filedata, &directives_list); err != nil {
				     //http.StatusNotAcceptable
				     fmt.Fprintf(w, "{\"status\": \"error\", \"message\": \"failed to parse json\"}")
				     return
				   }
				*/

				/*
				   dtype, ok := directives_list["type"].(string)
				   if !ok {
				     fmt.Fprintf(w, "{\"status\": \"error\", \"message\": \"invalid directive type\"}")
				     return
				   }
				*/

				// now publish it
				nc.Publish(dtype, filedata)
				resp_data := map[string]interface{}{
					"status":     "ok",
					"filename":   hdr.Filename,
					"directives": directives_list,
				}
				resp, _ := json.Marshal(resp_data)
				fmt.Fprintf(w, string(resp))
				return
			}
		}
		//fmt.Fprintf(w, "{\"status\": \"NOT_IMPLEMENTED\"}")
		//http.StatusNotAcceptable
		fmt.Fprintf(w, "{\"status\": \"error\", \"message\": \"no file specified\"}")
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Cache-control", "public, max-age=0")
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Server", "Brigade/"+VERSION)
		/*
		   err := server.renderFile(w, r.URL.Path)
		   if err != nil {
		     w.Header().Set("Content-Type", "text/html; charset=utf-8")
		     w.WriteHeader(http.StatusNotFound)
		     server.fn404(w, r)
		   }
		*/
		http.FileServer(http.Dir("./public"))

		if r.URL.Path[1:] == "" || r.URL.Path[1:] == "public/" {
			http.ServeFile(w, r, "./public/index.html")
		} else {
			http.ServeFile(w, r, r.URL.Path[1:])
		}

	})

	mux.HandleFunc("/api/configs/commander", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-control", "public, max-age=0")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "Brigade/"+VERSION)

		fmt.Fprintf(w, "{\"status\": \"ok\", \"message\": \"NOT IMPLEMENTED\"}")
	})

	mux.HandleFunc("/api/configs/node", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-control", "public, max-age=0")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "Brigade/"+VERSION)

		fmt.Fprintf(w, "{\"status\": \"ok\", \"message\": \"NOT IMPLEMENTED\"}")
	})

	mux.HandleFunc("/api/configs/directives", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-control", "public, max-age=0")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Server", "Brigade/"+VERSION)

		resp, _ := json.Marshal(directives_list)
		fmt.Fprintf(w, string(resp))
	})

	return mux
}

func checkInWithNodes(nc *nats.Conn, delay time.Duration) { // CONVERTED
	// Check-in with all nodes bing sendinga ping every X seconds

	for _ = range time.Tick(delay * time.Second) {
		nc.Publish("ping", []byte("checkin"))
	}
}
