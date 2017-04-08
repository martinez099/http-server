package main

import (
    "container/list"
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

// JSON format to persist timestamps
type JsonFormat struct {
    Timestamps []time.Time
}

// use a list to keep timestamps
var reqTs = list.New()

// mutex to synchronize list access
var lock = new(sync.Mutex)

// duration for how long to keep the timestamps
var oneMin, _ = time.ParseDuration("1m")

// filename to persist timestamps in JSON format
var filename = "requestTimestamps.json"

// port to listen on
var listenerPort = ":8080"

// HTTP handler function
func handleReq(w http.ResponseWriter, r *http.Request) {

    // get current timestamp
    var now = time.Now()

    // synchronize list access
    lock.Lock()

    // add current timestamp
    reqTs.PushBack(now)

    // remove outdated timestamps
    for {
        ts := reqTs.Front()
        if ts.Value.(time.Time).Before(now.Add(-oneMin)) {
            reqTs.Remove(ts)
        } else {
            break
        }
    }

    // get current amount of timestamps
    len := reqTs.Len()

    // release lock
    lock.Unlock()

    // write response
    io.WriteString(w, strconv.Itoa(len) + "\n")
}

func main() {

    // check if file exists and create it if not
    _, error := os.Stat(filename)
    if error != nil {
        if os.IsNotExist(error) {
            _, error = os.Create(filename)
            if error != nil {
                panic(error)
            }
        } else {
            panic(error)
        }
    }

    // read from file
    bytes, error := ioutil.ReadFile(filename)
    if error != nil {
        panic(error)
    }

    if len(bytes) > 0 {

        // decode timestamps from JSON
        var jsonData JsonFormat
        error = json.Unmarshal(bytes, &jsonData)
        if error != nil {
            panic(error)
        }

        // transfer timesamps from array slice to list
        for i := range jsonData.Timestamps {
            reqTs.PushBack(jsonData.Timestamps[i])
        }
    }

    // create server
    server := &http.Server {Addr: listenerPort}

    // set handler function to serve on root path
    http.HandleFunc("/", handleReq)

    // define and call goroutine to start HTTP server
    go func() {
        error = server.ListenAndServe()
        if error != nil {
            log.Printf("%s", error)
        }
    }()

    // create a channel for signal handler
    c := make(chan os.Signal, 1)

    // setup signal handler for SIGINT
    signal.Notify(c, os.Interrupt)

    // block until a SIGINT is received
    _ = <- c

    // shutdown server gracefully
    error = server.Shutdown(nil)
    if error != nil {
        panic(error)
    }

    // transfer timestamps from list to array slice for encoding
    var jsonData JsonFormat
    for ts := reqTs.Front(); ts != nil; ts = ts.Next() {
        jsonData.Timestamps = append(jsonData.Timestamps, ts.Value.(time.Time))
    }

    // encode timestamps into JSON
    bytes, error = json.Marshal(jsonData)
    if error != nil {
        panic(error)
    }

    // write to file
    error = ioutil.WriteFile(filename, bytes, 0666)
    if error != nil {
        panic(error)
    }

}
