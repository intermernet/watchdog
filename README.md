###Watchdog is a task timer which presents a URL to reset the timer###

It has some optional flags:

`-task` will set the command to run (enclose in quotes if using args)

`-time` will set the timer duration (use Go language time.Duration notation eg. 10h46m19s . defaults to 1s)

`-port` will set the TCP/IP port to listen on (defaults to port 8080)

`-local` controls if the web server is only listening on `localhost` (Set true or false, defaults to true)

`-stealth` controls if the web server hides the task and timer information (Set true or false, defaults to false)

`-onetime` controls if the program exits after running the task once (Set true or false, defaults to false)

`-reseturl` will set the url path to reset the timer (defaults to "/reset/")

`-restarturl` will set the url path to restart the timer after it expires (defaults to "/restart/")

####Example usage:####

`watchdog -task="/bin/sh ~/release_secret_documents.sh" -time=24h` will run the server on TCP/IP port 8080, will run the specified shell script in 1 day if not accessed at `http://localhost:8080/reset/` to reset the timer. Accessing the URL presents the timer and task information. The timer can be restarted after it expires by accessing `http://localhost:8080/restart/`

`watchdog -task="rm -Rf ~/secret_docs" -time=168h -port=80 -local=false -stealth=true -reseturl=/` will run the server on TCP/IP port 80, will delete the specified directory in 1 week if not accessed at `http://your.ip.address/` to reset the timer. The timer can be restarted after it expires by accessing `http://your.ip.address/restart/` . Accessing either URL will return a `404: Not Found` error.

`watchdog -task="shutdown now" -time=30m -port=1337 -stealth=true -onetime=true -reseturl=/r3537/` will run the server on TCP/IP port 1337, will shutdown the server in 30 minutes if not accessed at `http://localhost:1337/r3537/` to reset the timer. Accessing the URL will return a `404: Not Found` error. The program will exit after the timer expires.