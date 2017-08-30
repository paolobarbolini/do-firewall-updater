package main

import (
	"io/ioutil"
	"net"
	"net/http"
)

// Is there a better way to force golang to use ipv4 or ipv6?
type dialer struct {
	net.Dialer

	Network string
}

func (d *dialer) Dial(network, addr string) (net.Conn, error) {
	return d.Dialer.Dial(d.Network, addr)
}

var ipv4Client = http.Client{
	Transport: &http.Transport{
		Dial: (&dialer{
			Network: "tcp4",
		}).Dial,
	},
}

var ipv6Client = http.Client{
	Transport: &http.Transport{
		Dial: (&dialer{
			Network: "tcp6",
		}).Dial,
	},
}

func callIPApi(url string, ipv4 bool) (ip string, err error) {
	client := ipv4Client
	if !ipv4 {
		client = ipv6Client
	}

	resp, err := client.Get(url)
	if err != nil {
		return
	}

	data, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		return
	}

	ip = string(data)
	return
}
