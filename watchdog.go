package main

import (
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var task string
var duration string
var port int
var local bool
var stealth bool
var urlpath string

func init() {
	flag.StringVar(&task, "task", "echo Watchdog timed out...", "Command to execute")
	flag.StringVar(&duration, "time", "10s", "Time to wait")
	flag.IntVar(&port, "port", 8080, "TCP/IP Port to listen on")
	flag.BoolVar(&local, "local", true, "Listen on localhost only")
	flag.BoolVar(&stealth, "stealth", false, "No browser output")
	flag.StringVar(&urlpath, "urlpath", "/reset/", "URL Path to export")
}

func splitTaskString(task string) (string, []string, error) {
	argv := strings.Fields(task)
	c := argv[0]
	if len(argv) > 1 {
		return c, argv[1:], nil
	} else {
		return c, nil, nil
	}
}

type timerRecord struct {
	r string
	e error
}

type timedTask struct {
	task  string
	d     time.Duration
	timer *time.Timer
	rc    chan timerRecord
}

func (tt *timedTask) start() {
	defer close(tt.rc)
	tt.timer = time.NewTimer(tt.d)
	var tr timerRecord
	select {
	case <-tt.timer.C:
		task, argv, err := splitTaskString(tt.task)
		if err != nil {
			tr.r = ""
			tr.e = err
		}
		if argv == nil {
			out, err := exec.Command(task).Output()
			tr.r = string(out)
			tr.e = err
		} else {
			out, err := exec.Command(task, argv...).Output()
			tr.r = string(out)
			tr.e = err
		}
		tt.rc <- tr
	}
}

func makeHandlerFunc(fn func(http.ResponseWriter, *http.Request, *timedTask), tt *timedTask) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, tt)
	}
}

func resetHandler(w http.ResponseWriter, r *http.Request, tt *timedTask) {
	ct := time.Now()
	et := ct.Add(tt.d)
	if tt.timer.Reset(tt.d) {
		if stealth {
			http.NotFound(w, r)
		} else {
			fmt.Fprintf(w, "<!DOCTYPE html>\n<html>\n<head>\n<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" /><title>%s</title>\n</head>\n<body>", urlpath)
			fmt.Fprintf(w, "Timer reset at %s.<br>\nTimer expires at %s.<br>\nRunning \"%s\" when expired.<br>", html.EscapeString(ct.Format(time.RFC3339)), html.EscapeString(et.Format(time.RFC3339)), html.EscapeString(tt.task))
			fmt.Fprintf(w, "<a href=\"%s\">Reset Timer</a>", urlpath)
			fmt.Fprint(w, "</body>\n</html>")
		}
	}
}

func listen(rc chan timerRecord) {
	for tr := range rc {
		if tr.e != nil {
			log.Println("Error: ", tr.e)
		} else if tr.r != "" {
			log.Println(tr.r)
		}
		log.Fatal("Exiting...")
	}
}

func main() {
	flag.Parse()
	p := strconv.Itoa(port)
	addr := "localhost:" + p
	if local != true {
		addr = ":" + p
	}
	if !strings.HasPrefix(urlpath, "/") {
		urlpath = "/" + urlpath
	}
	if !strings.HasSuffix(urlpath, "/") {
		urlpath = urlpath + "/"
	}
	d, err := time.ParseDuration(duration)
	if err != nil {
		log.Fatal(err)
	}
	rc := make(chan timerRecord)
	tt := timedTask{task, d, nil, rc}
	go tt.start()
	go listen(rc)
	http.HandleFunc(urlpath, makeHandlerFunc(resetHandler, &tt))
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe: " + err.Error())
	}
}
