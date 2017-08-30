package main

import (
	"encoding/json"
	"errors"
	"flag"
	"io/ioutil"
	"os"
	"reflect"

	"gopkg.in/digitalocean/godo.v1"

	"log"
)

func loadOldIps() (oldIps []string, err error) {
	f, ok := os.Open("old_ips.json")
	if ok != nil {
		log.Println("Failed to read the old_ips.json file, new installation?")
		return
	}
	defer f.Close()

	var b []byte
	b, err = ioutil.ReadAll(f)
	if err != nil {
		return
	}

	err = json.Unmarshal(b, &oldIps)
	return
}

func loadNewIps(ipAPI string) (newIps []string, err error) {
	ipv4, ipv4err := callIPApi(ipAPI, true)
	ipv6, ipv6err := callIPApi(ipAPI, false)

	if ipv4err != nil && ipv6err != nil {
		err = errors.New("Failed to load the ipv4 and the ipv6")
		return
	}

	if ipv4err != nil {
		log.Printf("Failed to get the ipv4 address, is this machine ipv6 only? %s", ipv4err)
	} else {
		newIps = append(newIps, ipv4)
		log.Printf("Current ipv4 address: %s", ipv4)
	}

	if ipv6err != nil {
		log.Printf("Failed to get the ipv6 address, is this machine ipv4 only? %s", ipv6err)
	} else {
		newIps = append(newIps, ipv6)
		log.Printf("Current ipv6 address: %s", ipv6)
	}

	return
}

func saveNewIps(newIps []string) (err error) {
	f, err := os.Create("old_ips.json")
	if err != nil {
		return
	}
	defer f.Close()

	b, err := json.Marshal(newIps)
	if err != nil {
		return
	}

	f.Write(b)
	return
}

func main() {
	var apiToken, ipAPI, firewallName, firewallID string

	flag.StringVar(&apiToken, "token", "", "The digitalocean api token")
	flag.StringVar(&ipAPI, "ip-api", "http://v4v6.ipv6-test.com/api/myip.php", "The url of the ip api")
	flag.StringVar(&firewallName, "firewall-name", "", "The name of the firewall")
	flag.StringVar(&firewallID, "firewall-id", "", "The id of the firewall")

	flag.Parse()

	if apiToken == "" {
		log.Fatal("You must specify the --token")
	}

	if firewallName == "" && firewallID == "" {
		log.Fatal("You must specify the --firewall-name or the --firewall-id")
	}

	oldIps, err := loadOldIps()
	if err != nil {
		log.Fatal(err)
	}

	newIps, err := loadNewIps(ipAPI)
	if err != nil {
		log.Fatal(err)
	}

	if reflect.DeepEqual(oldIps, newIps) {
		log.Println("The ips didn't change.")
		return
	}

	client := newClient(apiToken)

	var firewall *godo.Firewall
	if firewallID != "" {
		firewall, err = findFirewallByID(client, firewallID)
	} else {
		firewall, err = findFirewallByName(client, firewallName)
	}

	if err != nil {
		log.Fatal(err)
	}

	// Small hack to fix inconsistencies in the digitalocean api
	firewall.InboundRules, firewall.OutboundRules = fixInboundOutboundRules(firewall.InboundRules, firewall.OutboundRules)

	for _, rule := range firewall.InboundRules {
		removed := false

		for i, address := range rule.Sources.Addresses {
			if address == oldIps[0] || (len(oldIps) > 1 && address == oldIps[1]) {
				removed = true
				rule.Sources.Addresses = append(rule.Sources.Addresses[:i], rule.Sources.Addresses[i+1:]...)
			}
		}

		if removed {
			rule.Sources.Addresses = append(rule.Sources.Addresses, newIps[:]...)
		}
	}

	fr := &godo.FirewallRequest{
		Name:          firewall.Name,
		InboundRules:  firewall.InboundRules,
		OutboundRules: firewall.OutboundRules,
		DropletIDs:    firewall.DropletIDs,
		Tags:          firewall.Tags,
	}

	err = updateFirewall(client, firewall.ID, fr)
	if err != nil {
		log.Fatal(err)
	}

	err = saveNewIps(newIps)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Firewall rules updated successfully.")
}
