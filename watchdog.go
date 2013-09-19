/*
Copyright Mike Hughes 2013 (intermernet AT gmail DOT com)

Watchdog is a task timer with web based reset / restart written in Go.
It acts as a dead man's switch for a defined task. (http://en.wikipedia.org/wiki/Dead_man%27s_switch)

LICENSE: BSD 3-Clause License (see http://opensource.org/licenses/BSD-3-Clause)
*/
package main

import (
	"errors"
	"flag"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
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
	flag.DurationVar(&duration, "time", 0*time.Second, "Time to wait. REQUIRED!\n   (-time Example 10h5m46s\n   See http://golang.org/pkg/time/#ParseDuration for formatting rules)")
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

func addSlashes(s string) string {
	if !strings.HasPrefix(s, "/") {
		s = "/" + s
	}
	if !strings.HasSuffix(s, "/") {
		s = s + "/"
	}
	return s
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

func makeHandlerFunc(fn func(http.ResponseWriter, *http.Request, *timedTask), tt *timedTask) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fn(w, r, tt)
	}
}

func resetHandler(w http.ResponseWriter, r *http.Request, tt *timedTask) {
	ct := time.Now()
	et := ct.Add(tt.d)
	if redirurl != "" {
		tt.timer.Reset(tt.d)
		http.Redirect(w, r, redirurl, http.StatusFound)
	} else if stealth {
		tt.timer.Reset(tt.d)
		http.NotFound(w, r)
	} else {
		fmt.Fprintf(w, "<!DOCTYPE html>\n<html>\n<head>\n<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" />\n<title>%s</title>\n</head>\n<body>\n", reseturl)
		if tt.timer.Reset(tt.d) {
			fmt.Fprintf(w, "Timer reset at %s.<br>\nTimer expires at %s.<br>\nRunning \"%s\" when expired.<br>\n", html.EscapeString(ct.Format(time.RFC3339)), html.EscapeString(et.Format(time.RFC3339)), html.EscapeString(tt.task))
			fmt.Fprintf(w, "<a href=\"%s\">Reset Timer</a>\n", reseturl)
		} else {
			fmt.Fprint(w, "Timer expired.<br>\n")
			fmt.Fprintf(w, "<a href=\"%s\">Restart Timer</a>\n", restarturl)
		}
		fmt.Fprint(w, "</body>\n</html>")
	}
}

func restartHandler(w http.ResponseWriter, r *http.Request, tt *timedTask) {
	go tt.start()
	ct := time.Now()
	et := ct.Add(tt.d)
	if redirurl != "" {
		http.Redirect(w, r, redirurl, http.StatusFound)
	} else if stealth {
		http.NotFound(w, r)
	} else {
		fmt.Fprintf(w, "<!DOCTYPE html>\n<html>\n<head>\n<meta http-equiv=\"Content-Type\" content=\"text/html; charset=UTF-8\" />\n<title>%s</title>\n</head>\n<body>\n", restarturl)
		fmt.Fprintf(w, "Timer restarted at %s.<br>\nTimer expires at %s.<br>\nRunning \"%s\" when expired.<br>\n", html.EscapeString(ct.Format(time.RFC3339)), html.EscapeString(et.Format(time.RFC3339)), html.EscapeString(tt.task))
		fmt.Fprintf(w, "<a href=\"%s\">Reset Timer</a>\n", reseturl)
		fmt.Fprint(w, "</body>\n</html>")
	}
}

func listen(rc chan timerRecord, oc chan string, ec chan error) {
	for tr := range rc {
		if tr.e != nil {
			ec <- tr.e
		} else if tr.r != "" {
			oc <- tr.r
		}
		if onetime {
			ec <- errors.New("Exiting...")
		}
	}
}

func launch(addr string, tt *timedTask, ec chan error) {
	http.HandleFunc(reseturl, makeHandlerFunc(resetHandler, tt))
	if !onetime {
		http.HandleFunc(restarturl, makeHandlerFunc(restartHandler, tt))
	}
	if err := http.ListenAndServe(addr, nil); err != nil {
		ec <- errors.New("ListenAndServe: " + err.Error())
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
		log.Printf("%s\n", errtxt)
		flag.Usage()
		os.Exit(1)
	}
	addr := ":" + strconv.Itoa(port)
	if local == true {
		addr = "localhost" + addr
	}
	reseturl = addSlashes(reseturl)
	restarturl = addSlashes(restarturl)
	ur, err := url.Parse(redirurl)
	if err != nil {
		log.Fatal(err)
	}
	redirurl = ur.String()
	rc := make(chan timerRecord)
	tt := timedTask{task, duration, nil, rc}
	oc := make(chan string)
	ec := make(chan error)
	defer close(rc)
	defer close(oc)
	defer close(ec)
	go tt.start()
	go listen(rc, oc, ec)
	go launch(addr, &tt, ec)
	for {
		select {
		case out := <-oc:
			log.Println(out)
		case err := <-ec:
			log.Fatal(err.Error())
		}
	}
}
