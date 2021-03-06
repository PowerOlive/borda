package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/getlantern/borda"
	"github.com/getlantern/borda/report"
	"github.com/getlantern/golog"
	"github.com/getlantern/tlsdefaults"
	"github.com/getlantern/zenodb/rpc"
	"github.com/vharitonsky/iniflags"
)

var (
	log = golog.LoggerFor("borda")

	httpsaddr   = flag.String("httpsaddr", ":443", "The address at which to listen for HTTPS connections")
	reportsaddr = flag.String("reportsaddr", "localhost:14443", "The address at which to listen for HTTPS connections to the reports")
	cliaddr     = flag.String("cliaddr", "localhost:17712", "The address at which to listen for gRPC cli connections, defaults to localhost:17712")
	pprofAddr   = flag.String("pprofaddr", "localhost:4000", "if specified, will listen for pprof connections at the specified tcp address")
	pkfile      = flag.String("pkfile", "pk.pem", "Path to the private key PEM file")
	certfile    = flag.String("certfile", "cert.pem", "Path to the certificate PEM file")
)

func main() {
	iniflags.Parse()

	if *pprofAddr != "" {
		go func() {
			log.Debugf("Starting pprof page at http://%s/debug/pprof", *pprofAddr)
			if err := http.ListenAndServe(*pprofAddr, nil); err != nil {
				log.Error(err)
			}
		}()
	}
	s, db, err := borda.TDBSave("zenodata", "schema.yaml")
	if err != nil {
		log.Fatalf("Unable to initialize tdb: %v", err)
	}

	rl, err := tlsdefaults.Listen(*reportsaddr, *pkfile, *certfile)
	if err != nil {
		log.Fatalf("Unable to listen for reports: %v", err)
	}
	fmt.Fprintf(os.Stdout, "Listening for report connections at %v\n", rl.Addr())
	r := &report.Handler{DB: db}
	go func() {
		serverErr := http.Serve(rl, r)
		if serverErr != nil {
			log.Errorf("Error serving reports: %v", serverErr)
		}
	}()

	hl, err := tlsdefaults.Listen(*httpsaddr, *pkfile, *certfile)
	if err != nil {
		log.Fatalf("Unable to listen HTTPS: %v", err)
	}
	fmt.Fprintf(os.Stdout, "Listening for HTTPS connections at %v\n", hl.Addr())

	if *cliaddr != "" {
		cl, err := net.Listen("tcp", *cliaddr)
		if err != nil {
			log.Fatalf("Unable to listen at cliaddr %v: %v", *cliaddr, err)
		}
		go rpc.Serve(db, cl)
	}

	h := &borda.Handler{Save: s}
	go h.Report()
	serverErr := http.Serve(hl, h)
	if serverErr != nil {
		log.Fatalf("Error serving HTTPS: %v", serverErr)
	}
}
