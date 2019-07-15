package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudflare/cloudflare-go"
	"github.com/miekg/dns"
)

const defaultTarget = "myip.opendns.com"
const defaultServer = "resolver1.opendns.com"
const defaultCachePath = "/tmp/.dynamic-dns-cloudflare.cache"

type DDNS struct {
	target       string
	server       string
	cachePath    string
	domainName   string
	zoneName     string
	pollInterval time.Duration
	cf           *cloudflare.API
}

func (d *DDNS) resolveIP() (*dns.A, error) {
	c := dns.Client{}
	m := dns.Msg{}
	m.SetQuestion(d.target+".", dns.TypeA)

	r, _, err := c.Exchange(&m, d.server+":53")
	if err != nil {
		return nil, fmt.Errorf("DNS query failed: %v", err)
	}

	if len(r.Answer) == 0 {
		return nil, fmt.Errorf("no DNS results")
	}

	for _, ans := range r.Answer {
		record, ok := ans.(*dns.A)
		if !ok {
			continue
		}

		return record, nil
	}

	return nil, fmt.Errorf("Unexpected DNS Result, no usable records: %#v", r.Answer)
}

func (d *DDNS) isIPChanged(ip *dns.A) (bool, error) {
	content, err := ioutil.ReadFile(d.cachePath)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("unexpected error reading cache: %v", err)
	}

	if string(content) == ip.String() {
		return false, nil
	}

	content = []byte(ip.String())
	err = ioutil.WriteFile(d.cachePath, content, 0644)
	if err != nil {
		return true, fmt.Errorf("unexpected error writing cache: %v", err)
	}

	return true, nil
}

func (d *DDNS) updateRecord(ip *dns.A) error {
	id, err := d.cf.ZoneIDByName(d.zoneName)
	if err != nil {
		return err
	}

	records, err := d.cf.DNSRecords(id, cloudflare.DNSRecord{Name: d.domainName})
	if err != nil {
		return err
	}

	if len(records) != 1 {
		return fmt.Errorf("Incorrect records response, expected 1, got: %d", len(records))
	}

	record := records[0]
	record.Content = ip.A.String()

	return d.cf.UpdateDNSRecord(id, record.ID, record)
}

func (d *DDNS) Update() error {
	ip, err := d.resolveIP()
	if err != nil {
		return err
	}

	changed, err := d.isIPChanged(ip)
	if err != nil {
		return err
	}

	if !changed {
		// nothing to do here
		return nil
	}

	log.Printf("[INFO] IP Address changed: %v", ip.A.String())

	return d.updateRecord(ip)
}

func main() {
	log.SetFlags(log.Lshortfile)

	target := flag.String("target", defaultTarget, "lookup target")
	server := flag.String("server", defaultServer, "dns resolver")
	cachePath := flag.String("path", defaultCachePath, "path to cache")
	domainName := flag.String("domain", "", "domain name")
	zoneName := flag.String("zone", "", "zone name")
	pollInterval := flag.String("interval", "", "time interval to update dns records (e.g 12h)")
	flag.Parse()

	api, err := cloudflare.New(os.Getenv("CF_API_KEY"), os.Getenv("CF_API_EMAIL"))
	if err != nil {
		log.Fatalf("failed to construct cloudflare api client, are CF_API_KEY and CF_API_EMAIL set? err: %v", err)
	}

	if *domainName == "" {
		log.Fatal("-domain flag must be set")
	}
	if *zoneName == "" {
		log.Fatal("-zone flag must be set")
	}

	var interval time.Duration
	if *pollInterval != "" {
		var err error
		interval, err = time.ParseDuration(*pollInterval)
		if err != nil {
			log.Fatalf("failed to parse duration (%s): %v", *pollInterval, err)
		}
	}

	ddns := &DDNS{
		target:       *target,
		server:       *server,
		cachePath:    *cachePath,
		domainName:   *domainName,
		zoneName:     *zoneName,
		pollInterval: interval,
		cf:           api,
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-timer.C:
			err := ddns.Update()
			log.Printf("[ERROR] Failed to update records: %v", err)
			if ddns.pollInterval != 0 {
				timer.Reset(ddns.pollInterval)
			} else {
				exitCode := 0
				if err != nil {
					exitCode = 1
				}
				os.Exit(exitCode)
			}
		case <-sigs:
			log.Print("Shutting down due to signal")
			os.Exit(0)
		}
	}
}
