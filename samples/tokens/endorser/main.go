/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/hyperledger/fabric-samples/token-sdk/common"
	"github.com/hyperledger/fabric-samples/token-sdk/endorser/routes"
	"github.com/hyperledger/fabric-samples/token-sdk/endorser/service"
)

func main() {
	// Flags
	cwd, _ := os.Getwd()
	pth := flag.String("conf", cwd, "the directory that contains the core.yaml configuration file")
	port := flag.String("port", "9000", "the API port for the application")
	flag.Parse()

	// Fabric smart client
	fsc, err := common.StartFSC(*pth, path.Join(*pth, "data"))
	if err != nil {
		log.Fatal(err)
	}

	// Simple web server
	sh := routes.NewStrictHandler(routes.NewServer(service.NewFSC(fsc)), []routes.StrictMiddlewareFunc{})
	h := common.WithAnyCORS(routes.HandlerFromMux(sh, http.NewServeMux()))
	s := &http.Server{
		Handler: h,
		Addr:    net.JoinHostPort("0.0.0.0", *port),
	}
	go s.ListenAndServe()

	// Stop
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	<-stop
	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*10)
	defer cancel()
	s.Shutdown(ctx)
	fsc.Stop()
}
