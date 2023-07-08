package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	client "github.com/alex-dna-tech/milvus-client"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var server string

func init() {
	flag.StringVar(
		&server, "s", os.Getenv("MDB_SERVER"),
		"connection string (e.g. 127.0.0.1:19530) or use MDB_SERVER environment varaiable",
	)
}

func main() {
	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime|log.Lshortfile)
	if server == "" {
		logger.Println("'s' key and MDB_SERVER environment variables are not set, using localhost:19530")
		server = "localhost:19530"
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	client, err := client.New(ctx, server)
	if err != nil {
		logger.Fatal("failed to connect to Milvus: ", err.Error())
	}
	defer client.Close()

	//Create a new instance of the milvus collector and
	//register it with the prometheus client.
	m := NewMilvusCollector(client)
	prometheus.MustRegister(m)

	//This section will start the HTTP server and expose
	//any metrics on the /metrics endpoint.
	http.Handle("/metrics", promhttp.Handler())
	logger.Println("Beginning to serve on port :8080")
	logger.Fatal(http.ListenAndServe(":8080", nil))
}
