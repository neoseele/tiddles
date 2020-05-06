package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/neoseele/tiddles/pkg/db"
	"github.com/neoseele/tiddles/pkg/dns"
	"github.com/neoseele/tiddles/pkg/dump"
	g "github.com/neoseele/tiddles/pkg/grpc"
	"github.com/neoseele/tiddles/pkg/probe"
	"github.com/neoseele/tiddles/pkg/stress"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	pb "google.golang.org/grpc/examples/helloworld/helloworld"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	mgo "gopkg.in/mgo.v2"

	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"go.opencensus.io/plugin/ocgrpc"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.opencensus.io/zpages"
)

// Index ...
func Index(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	host, _ := os.Hostname()
	version := "1.0.0"
	if fromEnv := os.Getenv("VERSION"); fromEnv != "" {
		version = fromEnv
	}

	fmt.Fprintf(w, "Hello, world!\n")
	fmt.Fprintf(w, "Version: %s\n", version)
	fmt.Fprintf(w, "Hostname: %s\n", host)

	requestDump, err := httputil.DumpRequest(r, true)
	if err != nil {
		fmt.Fprint(w, err.Error())
	} else {
		fmt.Fprintf(w, "\n== Header ==\n")
		fmt.Fprint(w, string(requestDump))
	}
}

// NotFound 404 handler
func NotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("404 - page not found...\n"))
}

// Error sample error
func Error(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "500 - a horrible error emerged...\n",
		http.StatusInternalServerError)
}

// logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return handlers.CombinedLoggingHandler(os.Stdout, next)
}

// start server
func runServer(router *mux.Router, httpPort string, httpsPort string, grpcPort string, zpagesPort string, tlsCert string, tlsKey string) chan error {
	errs := make(chan error)

	// Starting HTTP server
	go func() {
		log.Printf("Staring HTTP service on %s ...", httpPort)
		if err := http.ListenAndServe(":"+httpPort, &ochttp.Handler{
			Handler:     router,
			Propagation: &propagation.HTTPFormat{},
		}); err != nil {
			errs <- err
		}
	}()

	if tlsCert != "" && tlsKey != "" {
		// Starting HTTPS server
		go func() {
			log.Printf("Staring HTTPS service on %s ...", httpsPort)
			if err := http.ListenAndServeTLS(":"+httpsPort, tlsCert, tlsKey, &ochttp.Handler{
				Handler:     router,
				Propagation: &propagation.HTTPFormat{},
			}); err != nil {
				errs <- err
			}
		}()
	}

	// Starting grpc server
	go func() {
		log.Printf("Staring gRPC service on %s ...", grpcPort)
		lis, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			log.Fatalf("failed to listen: %v", err)
		}

		var grpcOptions []grpc.ServerOption
		grpcOptions = append(grpcOptions, grpc.StatsHandler(&ocgrpc.ServerHandler{}))

		if tlsCert != "" && tlsKey != "" {
			creds, err := credentials.NewServerTLSFromFile(tlsCert, tlsKey)
			if err != nil {
				log.Fatalf("Invalid TLS credentials: %v\n", err)
			}
			log.Printf("Using server certificate %v to construct TLS credentials", tlsCert)
			log.Printf("Using server key %v to construct TLS credentials", tlsKey)
			grpcOptions = append(grpcOptions, grpc.Creds(creds))
		}

		s := grpc.NewServer(grpcOptions...)
		pb.RegisterGreeterServer(s, &g.GreeterServer{})
		healthpb.RegisterHealthServer(s, &g.HealthServer{})
		if err := s.Serve(lis); err != nil {
			errs <- err
		}
	}()

	go func() {
		mux := http.NewServeMux()
		zpages.Handle(mux, "/")

		addr := ":" + zpagesPort
		log.Printf("Staring zPages HTTP service on %s ...", zpagesPort)
		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Failed to serve zPages")
			errs <- err
		}
	}()

	return errs
}

func initStats(exporter *stackdriver.Exporter) {
	view.SetReportingPeriod(60 * time.Second)
	view.RegisterExporter(exporter)
	if err := view.Register(ocgrpc.DefaultServerViews...); err != nil {
		log.Printf("Error registering default server views")
	} else {
		log.Printf("Registered default server views")
	}
}

func initStackdriverTracing() {
	for i := 1; i <= 3; i++ {
		exporter, err := stackdriver.NewExporter(stackdriver.Options{
			// ProjectID: "nmiu-play",
		})
		if err != nil {
			log.Printf("failed to initialize Stackdriver exporter: %+v", err)
		} else {
			trace.RegisterExporter(exporter)
			trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
			log.Printf("registered Stackdriver tracing")

			// Register the views to collect server stats.
			initStats(exporter)
			return
		}
		d := time.Second * 10 * time.Duration(i)
		log.Printf("sleeping %v to retry initializing Stackdriver exporter", d)
		time.Sleep(d)
	}
	log.Printf("could not initialize Stackdriver exporter after retrying, giving up")
}

// main function to boot up everything
func main() {

	// set backend if the flag is set
	backend := flag.String("backend", "", "Specify a backend url to ping [localhost:80] (default: none)")
	httpPort := flag.String("http-port", "80", "Specify a http port (default: 80)")
	httpsPort := flag.String("https-port", "443", "Specify a https port (default: 443)")
	grpcPort := flag.String("grpc-port", "50000", "Specify a grpc port (default: 50000)")
	zpagesPort := flag.String("zpages-port", "8888", "Specify a http port for zpages endpoints (default: 8888)")
	cert := flag.String("cert", "", "Specify a TLS cert file (default: none)")
	key := flag.String("key", "", "Specify a TLS key file (default: none)")
	grpcBeAddr := flag.String("grpc-backend", "", "Specify a grpc backend address [localhost:50000] (default: none)")
	clientOnly := flag.Bool("client-only", false, "Run as client (default: false")
	doTrace := flag.Bool("trace", false, "Enable Stackdriver Tracing (default: false)")
	flag.Parse()

	// run as client
	if *clientOnly {
		g.PingBackend(context.Background(), *grpcBeAddr, *cert)
		os.Exit(0)
	}

	// enable Stackdriver Tracing
	if *doTrace {
		go initStackdriverTracing()
	}

	// set mongodb connection string if specified via env
	mongoDbURL := os.Getenv("MONGODB_URL")

	router := mux.NewRouter()
	router.NotFoundHandler = loggingMiddleware(http.HandlerFunc(NotFound))
	router.Use(loggingMiddleware)
	router.HandleFunc("/", Index)
	router.HandleFunc("/error", Error).Methods("GET")

	// db
	if mongoDbURL != "" {
		session, err := mgo.Dial(mongoDbURL)
		if err != nil {
			panic(err)
		}
		defer session.Close()
		db.Init(session)

	} else {
		db.Init(nil)
	}
	router.HandleFunc("/db", db.GetAll).Methods("GET")
	router.HandleFunc("/db/{id:[0-9]+}", db.Get).Methods("GET")
	router.HandleFunc("/db/{id:[0-9]+}", db.Create).Methods("POST")
	router.HandleFunc("/db/{id:[0-9]+}", db.Update).Methods("PUT")
	router.HandleFunc("/db/{id:[0-9]+}", db.Delete).Methods("DELETE")

	// stress
	router.HandleFunc("/stress/{type}", stress.Run).Methods("GET")

	// dns
	router.HandleFunc("/dns", dns.Run).Methods("GET")

	// probe
	router.HandleFunc("/health", probe.Health).Methods("GET")
	router.HandleFunc("/liveness", probe.Liveness).Methods("GET")
	router.HandleFunc("/readiness", probe.Readiness).Methods("GET")
	router.HandleFunc("/ping-backend", func(w http.ResponseWriter, r *http.Request) {
		probe.PingBackend(w, r, *backend)
	}).Methods("GET")
	router.HandleFunc("/ping-backend-with-db", func(w http.ResponseWriter, r *http.Request) {
		probe.PingBackend(w, r, *backend+"/db")
		// probe.PingGRPCBackend(w, r, *grpcBeAddr, *cert)
	}).Methods("GET")
	router.HandleFunc("/ping-grpc-backend", func(w http.ResponseWriter, r *http.Request) {
		probe.PingGRPCBackend(w, r, *grpcBeAddr, *cert)
	}).Methods("GET")

	// dump
	router.HandleFunc("/dump/", dump.GetAll).Methods("GET")
	router.HandleFunc("/dump/{name}", dump.GetObj).Methods("GET")

	// log.Fatal(http.ListenAndServe(":"+port, router))
	errs := runServer(router, *httpPort, *httpsPort, *grpcPort, *zpagesPort, *cert, *key)

	// This will run forever until channel receives error
	select {
	case err := <-errs:
		log.Printf("Could not start serving service due to (error: %s)", err)
	}
}
