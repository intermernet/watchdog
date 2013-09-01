###Watchdog is a task timer which presents a URL to reset the timer###

It has 6 optional flags:

`-task` will set the command to run (enclose in quotes if using args)

`-time` will set the timer duration (use Go language time.Duration notation eg. 10h46m19s . defaults to 1s)

`-port` will set the TCP/IP port to listen on (defaults to port 8080)

`-local` controls if the web server is only listening on `localhost` (Set true or false, defaults to true)

`-stealth` controls if the web server hides the task and timer information (Set true or false, defaults to false)

`-urlpath` will set the url path to export (defaults to "/reset/")

####Example usage:####

`watchdog -task="/bin/sh ~/release_secret_documents.sh" -time=24h` will run the server on TCP/IP port 8080, will run the specified shell script in 1 day if not accessed at `http://localhost:8080/reset/` to reset the timer. Accessing the URL presents the timer and task information.

`watchdog -task="rm -Rf ~/secret_docs" -time=168h -port=80 -local=false -stealth=true -urlpath=/` will run the server on TCP/IP port 80, will delete the specified directory in 1 week if not accessed at `http://your.domain/` to reset the timer. Accessing the URL will return a `404: Not Found` error.