package main

import (
	"errors"
	"flag"
	"fmt"
	"github.com/digitalocean/go-libvirt"
	"github.com/gorilla/mux"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	bindAddr = flag.String("b", ":8084", "API bind address:port")
)

var lvrt *libvirt.Libvirt

func toUuid(uuid string, flags libvirt.ConnectListAllDomainsFlags) (error, libvirt.UUID) {
	targetUuid := strings.ReplaceAll(uuid, "-", "")

	domains, _, err := lvrt.ConnectListAllDomains(1024, flags)
	if err != nil {
		log.Fatalf("failed to retrieve domains: %v", err)
	}

	for _, domain := range domains {
		currentUuid := fmt.Sprintf("%x", domain.UUID)
		if currentUuid == targetUuid {
			return nil, domain.UUID
		}
	}

	return errors.New("unable to find VM"), libvirt.UUID{}
}

func hShutdownVM(w http.ResponseWriter, r *http.Request) {
	err, vmUuid := toUuid(mux.Vars(r)["uuid"], libvirt.ConnectListDomainsActive)
	if err != nil {
		fmt.Fprintf(w, "Error: (query) "+err.Error())
	}

	err = lvrt.DomainShutdown(libvirt.Domain{UUID: vmUuid})
	if err != nil {
		fmt.Fprintf(w, "Error: "+err.Error())
		return
	}
	fmt.Fprintf(w, "Success: shutdown complete")
	return
}

func hResetVM(w http.ResponseWriter, r *http.Request) {
	err, vmUuid := toUuid(mux.Vars(r)["uuid"], libvirt.ConnectListDomainsActive)
	if err != nil {
		fmt.Fprintf(w, "Error: (query) "+err.Error())
	}

	err = lvrt.DomainReset(libvirt.Domain{UUID: vmUuid}, 0)
	if err != nil {
		fmt.Fprintf(w, "Error: "+err.Error())
		return
	}
	fmt.Fprintf(w, "Success: reset complete")
	return
}

func hRebootVM(w http.ResponseWriter, r *http.Request) {
	err, vmUuid := toUuid(mux.Vars(r)["uuid"], libvirt.ConnectListDomainsActive)
	if err != nil {
		fmt.Fprintf(w, "Error: (query) "+err.Error())
	}

	err = lvrt.DomainReboot(libvirt.Domain{UUID: vmUuid}, libvirt.DomainRebootDefault)
	if err != nil {
		fmt.Fprintf(w, "Error: "+err.Error())
		return
	}
	fmt.Fprintf(w, "Success: reboot complete")
	return
}

func main() {
	c, err := net.DialTimeout("tcp", "10.0.100.1:16509", 2*time.Second)
	if err != nil {
		log.Fatalf("failed to dial libvirt: %v", err)
	}

	lvrt = libvirt.New(c)
	if err := lvrt.Connect(); err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	log.Printf("connected to %s\n", c.RemoteAddr())

	v, err := lvrt.ConnectGetLibVersion()
	if err != nil {
		log.Fatalf("failed to retrieve libvirt version: %v", err)
	}
	fmt.Println("Version:", v)

	// Disconnect
	defer func() {
		if err := lvrt.Disconnect(); err != nil {
			log.Fatalf("failed to disconnect: %v", err)
		}
	}()

	// API
	r := mux.NewRouter()
	r.HandleFunc("/shutdown/{uuid}", hShutdownVM)
	r.HandleFunc("/reset/{uuid}", hResetVM)
	r.HandleFunc("/reboot/{uuid}", hRebootVM)
	http.Handle("/", r)

	log.Printf("Starting server on %s\n", *bindAddr)
	log.Fatal(http.ListenAndServe(*bindAddr, nil))
}
