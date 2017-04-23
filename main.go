package main

import (
    "io"
    "io/ioutil"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "os/signal"
    "strconv"
    "sync"
    "time"
)

// address HTTP server listens on
const ADDRESS = ":8080"

// filename to persist timestamps
const FILENAME = "requestTimestamps.json"

// duration to keep the timestamps
const DURATION = "1m"

// struct to hold the timestamps
type Counter struct {
    Timestamps []time.Time
    lock *sync.Mutex
}

// get amount of timestamps
func (c *Counter) Len() int {
    return len(c.Timestamps)
}

// add current timestamp
func (c *Counter) Inc(d time.Duration) {
    c.lock.Lock()

    // append current timestamp
    now := time.Now()
    c.Timestamps = append(c.Timestamps, now)

    for {
        if c.Timestamps[0].Before(now.Add(-d)) {
            // delete first timestamp
            copy(c.Timestamps[0:], c.Timestamps[1:])
            c.Timestamps[len(c.Timestamps)-1] = *new(time.Time)
            c.Timestamps = c.Timestamps[:len(c.Timestamps)-1]
        } else {
            break
        }
    }
    c.lock.Unlock()
}

// returns an HTTP handler function
func getHandler(c *Counter, d time.Duration) func(w http.ResponseWriter, r *http.Request) {
    return func (w http.ResponseWriter, r *http.Request) {

        // add current timestamp
        c.Inc(d)

        // get amount of timestamps
        len := c.Len()

        // write response
        io.WriteString(w, strconv.Itoa(len) + "\n")
    }
}

// error handler
func check(e error, isPanic bool) {
    if e != nil {
        log.Fatal("%s", e)
        if isPanic {
            panic(e)
        }
    }
}

func main() {

    // init counter
    counter := new(Counter)
    counter.lock = new(sync.Mutex)

    // check if file exists and create it if not
    _, error := os.Stat(FILENAME)
    if error != nil && os.IsNotExist(error) {
        _, error = os.Create(FILENAME)
    }
    check(error, true);

    // read timestamps from file
    bytes, error := ioutil.ReadFile(FILENAME)
    check(error, true)

    if len(bytes) > 0 {
        // decode timestamps from JSON format
        error = json.Unmarshal(bytes, &counter.Timestamps)
        check(error, true)
    }

    // create HTTP server
    server := &http.Server {Addr: ADDRESS}

    // parse duration
    duration, error := time.ParseDuration(DURATION)
    check(error, true)

    // set handler function to serve on root path
    http.HandleFunc("/", getHandler(counter, duration))

    // start HTTP server
    go func() {
        error = server.ListenAndServe()
        check(error, false)
    }()

    // create a channel for signal handler
    c := make(chan os.Signal, 1)

    // setup signal handler for SIGINT
    signal.Notify(c, os.Interrupt)

    // block until a SIGINT is received
    _ = <- c

    // shutdown HTTP server
    error = server.Shutdown(nil)
    check(error, false)

    // encode timestamps into JSON format
    bytes, error = json.Marshal(counter.Timestamps)
    check(error, true)

    // write timestamps to file
    error = ioutil.WriteFile(FILENAME, bytes, 0666)
    check(error, true)
}
