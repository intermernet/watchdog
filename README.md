###Watchdog is a task timer which presents a URL to reset the timer###

It has some flags, only the first 2 are required:

`-task` will set the command to run (enclose in quotes if using args) *REQUIRED*

`-time` will set the timer duration (use [Go language time.Duration notation](http://golang.org/pkg/time/#ParseDuration) eg. 10h46m19s .) *REQUIRED*

>>"A duration string is a possibly signed sequence of decimal numbers, each with optional fraction and a unit suffix, such as "300ms", "-1.5h" or "2h45m". Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h"."

`-port` will set the TCP/IP port to listen on (defaults to port 8080)

`-local` controls if the web server is only listening on `localhost` (Set true or false, defaults to true)

`-stealth` controls if the web server hides the task and timer information (Set true or false, defaults to false)

`-onetime` controls if the program exits after running the task once (Set true or false, defaults to false)

`-reseturl` will set the URL path to reset the timer (defaults to "/reset/")

`-restarturl` will set the URL path to restart the timer after it expires (defaults to "/restart/")

`-redirurl` will set the URL to redirect to after accessing either the reset or restart URLs. (defaults to nothing)

####Example usage:####

`watchdog -task=~/release_secret_documents.sh -time=24h` will run the server on TCP/IP port 8080, will run the specified shell script in 1 day if not accessed at `http://localhost:8080/reset/` to reset the timer. The timer can be restarted after it expires by accessing `http://localhost:8080/restart/` . Accessing either URL presents the timer and task information.

`watchdog -task="rm -Rf ~/secret_docs" -time=168h -port=80 -local=false -stealth=true -reseturl=/` will run the server on TCP/IP port 80, will delete the specified directory in 1 week if not accessed at `http://your.ip.address/` to reset the timer. The timer can be restarted after it expires by accessing `http://your.ip.address/restart/` . Accessing either URL will return a `404: Not Found` error, but will still reset or restart the timer.

`watchdog -task="shutdown now" -time=30m -port=1337 -onetime=true -reseturl=/r3537/` will run the server on TCP/IP port 1337, will shutdown the server in 30 minutes if not accessed at `http://localhost:1337/r3537/` to reset the timer. The program will exit after the timer expires. Accessing the reset URL will present the timer and task information. Accessing the restart URL will return a `404: Not Found` error, and will have no function.

`watchdog -task="wakeonlan 13:37:de:ad:be:ef" -time=48h -redirurl=http://www.google.com/` will run the server on TCP/IP port 8080, will wake the computer with the specified MAC address in 2 days if not accessed at `http://localhost:8080/reset/` to reset the timer. The timer can be restarted after it expires by accessing `http://localhost:8080/restart/` . Accessing either URL will redirect to the URL specified by `-redirurl`, but will still reset or restart the timer.