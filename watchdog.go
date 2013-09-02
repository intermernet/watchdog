/*
Copyright Mike Hughes 2013 (intermernet AT gmail DOT com)

Watchdog is a task timer with web based reset / restart written in Go.
It acts as a dead man's switch for a defined task. (http://en.wikipedia.org/wiki/Dead_man%27s_switch)

LICENSE: BSD 3-Clause License (see http://opensource.org/licenses/BSD-3-Clause)
*/
package main

import (
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var task string
var duration time.Duration
var port int
var local bool
var stealth bool
var onetime bool
var reseturl string
var restarturl string
var redirurl string

func init() {
	flag.StringVar(&task, "task", "", "Command to execute. REQUIRED!")
	flag.DurationVar(&duration, "time", 0*time.Second, "Time to wait. REQUIRED!")
	flag.IntVar(&port, "port", 8080, "TCP/IP Port to listen on")
	flag.BoolVar(&local, "local", true, "Listen on localhost only")
	flag.BoolVar(&stealth, "stealth", false, "No browser output (defaults to false)")
	flag.BoolVar(&onetime, "onetime", false, "Run timer once only (defaults to false)")
	flag.StringVar(&reseturl, "reseturl", "/reset/", "URL Path to export")
	flag.StringVar(&restarturl, "restarturl", "/restart/", "URL Path to export")
	flag.StringVar(&redirurl, "redirurl", "", "URL Path to redirect to after reset / restart")
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

func (tt *timedTask) stop() {
	close(tt.rc)
}

func makeResetHandlerFunc(fn func(http.ResponseWriter, *http.Request, *timedTask), tt *timedTask) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, tt)
	}
}

func makeRestartHandlerFunc(fn func(http.ResponseWriter, *http.Request, *timedTask, chan timerRecord), tt *timedTask, rc chan timerRecord) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, tt, rc)
	}
}

func resetHandler(w http.ResponseWriter, r *http.Request, tt *timedTask) {
	ct := time.Now()
	et := ct.Add(tt.d)
	if redirurl != "" {
		if tt.timer.Reset(tt.d) {
			http.Redirect(w, r, redirurl, http.StatusFound)
		}
	} else if stealth {
		http.NotFound(w, r)
	} else {
		fmt.Fprintf(w, "<!DOCTYPE html>\n<html>\n<head>\n<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" /><title>%s</title>\n</head>\n<body>", reseturl)
		if tt.timer.Reset(tt.d) {
			fmt.Fprintf(w, "Timer reset at %s.<br>\nTimer expires at %s.<br>\nRunning \"%s\" when expired.<br>", html.EscapeString(ct.Format(time.RFC3339)), html.EscapeString(et.Format(time.RFC3339)), html.EscapeString(tt.task))
			fmt.Fprintf(w, "<a href=\"%s\">Reset Timer</a>", reseturl)
		} else {
			fmt.Fprint(w, "Timer expired.<br>\n")
			fmt.Fprintf(w, "<a href=\"%s\">Restart Timer</a>", restarturl)
		}
		fmt.Fprint(w, "</body>\n</html>")
	}
}

func restartHandler(w http.ResponseWriter, r *http.Request, tt *timedTask, rc chan timerRecord) {
	tt.rc = rc
	go tt.start()
	ct := time.Now()
	et := ct.Add(tt.d)
	if redirurl != "" {
		http.Redirect(w, r, redirurl, http.StatusFound)
	} else if stealth {
		http.NotFound(w, r)
	} else {
		fmt.Fprintf(w, "<!DOCTYPE html>\n<html>\n<head>\n<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" /><title>%s</title>\n</head>\n<body>", restarturl)
		fmt.Fprintf(w, "Timer restarted at %s.<br>\nTimer expires at %s.<br>\nRunning \"%s\" when expired.<br>", html.EscapeString(ct.Format(time.RFC3339)), html.EscapeString(et.Format(time.RFC3339)), html.EscapeString(tt.task))
		fmt.Fprintf(w, "<a href=\"%s\">Reset Timer</a>", reseturl)
		fmt.Fprint(w, "</body>\n</html>")
	}
}

func listen(rc chan timerRecord, onetime bool) {
	for tr := range rc {
		if tr.e != nil {
			log.Println("Error: ", tr.e)
		} else if tr.r != "" {
			log.Println(tr.r)
		}
		if onetime {
			log.Fatal("Exiting...")
		}
	}
}

func main() {
	flag.Parse()
	errtxt := ""
	if task == "" {
		errtxt += "\"task\" flag required\n"
	}
	if duration <= 0*time.Second {
		errtxt += "\"time\" flag required, and must be positive.\n"
	}
	if errtxt != "" {
		log.Fatal("\n", errtxt)
	}
	ur, err := url.Parse(redirurl)
	if err != nil {
		log.Fatal(err)
	}
	redirurl = ur.String()
	p := strconv.Itoa(port)
	addr := "localhost:" + p
	if local != true {
		addr = ":" + p
	}
	if !strings.HasPrefix(reseturl, "/") {
		reseturl = "/" + reseturl
	}
	if !strings.HasSuffix(reseturl, "/") {
		reseturl = reseturl + "/"
	}
	if !strings.HasPrefix(restarturl, "/") {
		restarturl = "/" + restarturl
	}
	if !strings.HasSuffix(restarturl, "/") {
		restarturl = restarturl + "/"
	}
	rc := make(chan timerRecord)
	tt := timedTask{task, duration, nil, rc}
	defer tt.stop()
	go tt.start()
	go listen(rc, onetime)
	http.HandleFunc(reseturl, makeResetHandlerFunc(resetHandler, &tt))
	if !onetime {
		http.HandleFunc(restarturl, makeRestartHandlerFunc(restartHandler, &tt, rc))
	}
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal("ListenAndServe: " + err.Error())
	}
}
