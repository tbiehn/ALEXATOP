/*
* Copyright 2019, Travis Biehn
* All rights reserved.
*
* This source code is licensed under the MIT license found in the
* LICENSE file in the root directory of this source tree.
*
 */
package main

import (
	"bufio"
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

var e = log.New(os.Stderr, "", 0)
var l = log.New(os.Stdout, "", 0)

var Threshold = flag.Uint64("n", 50, "Top -n[umber] results")
var Threads = flag.Int("threads", 200, "How many parallel resolver -threads")

//PROFESSIONAL MULTITHREADING
var found = uint64(0)

func main() {
	e.Print(ban)
	dnsFile := flag.String("nameFile", "top1m.txt", "read names from specified -nameFile.")
	rangeURL := flag.String("rangeURL", "https://www.cloudflare.com/ips-v4", "read ranges from specified -rangeURL. (Try file://)")
	rangeURL2 := flag.String("rangeURL2", "https://www.cloudflare.com/ips-v6", "read ranges from specified -rangeURL2. -rangeURL2=\"\" to disable. (Try file://)")

	_ = dnsFile
	_ = rangeURL
	_ = rangeURL2

	flag.Parse()

	inFile, err := os.Open(*dnsFile)

	if err != nil {
		e.Print("[E] Failed to open file, ", *dnsFile)
		e.Fatal(err)
	}

	doSecond := true
	if strings.EqualFold("", *rangeURL2) {
		doSecond = false
	}

	r1, err := url.Parse(*rangeURL)
	if err != nil {
		e.Print("[E] Failed to parse rangeURL, ", *rangeURL)
		e.Fatal(err)

	}

	r2, err := url.Parse(*rangeURL)

	//Build net.IPNet array
	matchList := make([]*net.IPNet, 0, 500)

	if doSecond {
		r2, err = url.Parse(*rangeURL2)
		if err != nil {
			e.Print("[E] Failed to parse rangeURL2, ", *rangeURL2)
			e.Fatal(err)
		}
	}

	e.Print("Reading CIDRs from: ", r1.String())

	if strings.EqualFold("file", r1.Scheme) {
		//read local
		file, err := os.Open(r1.Path)
		if err != nil {
			e.Print("[E] Couldn't grab CIDRs from file, ", r1.Path)
			e.Fatal(err)
		}
		matchList = appendAll(matchList, file)

	} else {
		//read remote.
		res, err := http.Get(r1.String())
		if err != nil {
			e.Print("[E] Couldn't grab CIDRs from URL, ", r1.String())
			e.Fatal(err)
		}
		matchList = appendAll(matchList, res.Body)
	}

	if doSecond {
		e.Print("Reading CIDRs from: ", r2.String())
	}

	//check w/ doSecond
	if doSecond && strings.EqualFold("file", r2.Scheme) {
		//read local
		file, err := os.Open(r2.Path)
		if err != nil {
			e.Print("[E] Couldn't grab CIDRs from file, ", r2.Path)
			e.Fatal(err)
		}
		matchList = appendAll(matchList, file)

	} else if doSecond {
		//read remote.

		res, err := http.Get(r2.String())
		if err != nil {
			e.Print("[E] Couldn't grab CIDRs from URL, ", r1.String())
			e.Fatal(err)

		}
		matchList = appendAll(matchList, res.Body)

	}

	e.Print("Loaded ", len(matchList), " CIDRs")

	//YOLO
	jobs := make(chan AssessParcel, *Threads*5)

	for w := 1; w <= *Threads; w++ {
		go assessWorker(w, jobs)
	}
	var workerGroup sync.WaitGroup

	scanner := bufio.NewScanner(inFile)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		workerGroup.Add(1)

		name := scanner.Text()
		jobs <- AssessParcel{
			Name:      name,
			MatchList: matchList,
			Wg:        &workerGroup,
		}

	}

	close(jobs)
	workerGroup.Wait()

}

type AssessParcel struct {
	Name      string
	MatchList []*net.IPNet

	Wg *sync.WaitGroup
}

func assessWorker(id int, jobs <-chan AssessParcel) {
	for j := range jobs {
		assess(j.Name, j.MatchList, jobs)
		j.Wg.Done()
	}
}

func assess(name string, matchList []*net.IPNet, jobs <-chan AssessParcel) {
	e.Print("[I] Checking: ", name)

	aRecords, err := net.LookupIP(name)
	if err != nil {
		e.Print("[I] Failed to resolve IPs for ", name)
		e.Print(err)
	}

	inRange := false
	for _, cidr := range matchList {
		for _, ip := range aRecords {
			if cidr.Contains(ip) {
				e.Print("[I] Match for ", name, " ip: ", ip.String(), " in CIDR: ", cidr.String())
				inRange = true
				break
			}
		}
		if inRange {
			break
		}
	}
	if inRange {

		l.Print(name, " found.")

		//PROFESSIONAL MULTITHREADING
		results := atomic.AddUint64(&found, 1)

		if results >= *Threshold {

			//big brain time.
			for j := range jobs {
				j.Wg.Done()
			}

		}
	}
}

func appendAll(arr []*net.IPNet, read io.Reader) (out []*net.IPNet) {
	scanner := bufio.NewScanner(read)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		e.Print("[I] Parsing CIDR: ", scanner.Text())
		_, network, err := net.ParseCIDR(scanner.Text())
		if err != nil {
			e.Print("[I] Skipping non CIDR, ", scanner.Text(), "because; ")
			e.Print(err)
		}
		arr = append(arr, network)
	}
	return arr

}

var ban = `
   mm   m      mmmmmm m    m   mm  mmmmmmm  mmmm  mmmmm
   ##   #      #       #  #    ##     #    m"  "m #   "#
  #  #  #      #mmmmm   ##    #  #    #    #    # #mmm#"
  #mm#  #      #       m""m   #mm#    #    #    # #
 #    # #mmmmm #mmmmm m"  "m #    #   #     #mm#  #

dualuse.io - FINE DUAL USE TECHNOLOGIES
`
