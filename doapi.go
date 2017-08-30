package main

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/oauth2"

	"gopkg.in/digitalocean/godo.v1"
)

type tokenSource struct {
	AccessToken string
}

func (t *tokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func newClient(token string) *godo.Client {
	tokenSource := &tokenSource{
		AccessToken: token,
	}
	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	client := godo.NewClient(oauthClient)

	return client
}

// Hack to fix responses from the digitalocean api.
// The api returns the inbound and the outbound rules with 2 differences which are not documented
// and are not supported by any endpoint accepting inbound and outbound rules.
// This issue has already been reported in the support ticket #542033
func fixInboundOutboundRules(inboundRules []godo.InboundRule, outboundRules []godo.OutboundRule) ([]godo.InboundRule, []godo.OutboundRule) {
	for i, r := range inboundRules {
		if r.Protocol == "icmp" {
			// All of the firewall endpoints don't accept icmp rules with a "ports" field
			// Once this struct is converted to json the "ports" attribute won't be present because the tag contains "omitempty"
			r.PortRange = ""
			inboundRules[i] = r
			continue
		}

		if r.PortRange == "0" {
			// The api returns "0" instead of "all"
			// But all of the firewall endpoints accepting inbound and outbound rules don't allow the "ports" field to be set to "0"
			r.PortRange = "all"
			inboundRules[i] = r
		}
	}

	for i, r := range outboundRules {
		if r.Protocol == "icmp" {
			// All of the firewall endpoints don't accept icmp rules with a "ports" field
			// Once this struct is converted to json the "ports" attribute won't be present because the tag contains "omitempty"
			r.PortRange = ""
			outboundRules[i] = r
			continue
		}

		if r.PortRange == "0" {
			// The api returns "0" instead of "all"
			// But all of the firewall endpoints accepting inbound and outbound rules don't allow the "ports" field to be set to "0"
			r.PortRange = "all"
			outboundRules[i] = r
		}
	}

	return inboundRules, outboundRules
}

func findFirewallByName(client *godo.Client, name string) (firewall *godo.Firewall, err error) {
	ctx := context.TODO()
	options := &godo.ListOptions{
		PerPage: 200,
	}

	firewalls, _, err := client.Firewalls.List(ctx, options)
	if err != nil {
		return
	}

	for _, firewall := range firewalls {
		if !strings.EqualFold(firewall.Name, name) {
			continue
		}

		return &firewall, nil
	}

	err = errors.New("Firewall Not Found")
	return
}

func findFirewallByID(client *godo.Client, fID string) (firewall *godo.Firewall, err error) {
	ctx := context.TODO()

	firewall, _, err = client.Firewalls.Get(ctx, fID)
	return
}

func updateFirewall(client *godo.Client, fID string, fr *godo.FirewallRequest) (err error) {
	ctx := context.TODO()

	_, _, err = client.Firewalls.Update(ctx, fID, fr)
	return
}
