package gateway

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"sigs.k8s.io/external-dns/endpoint"
)

const defaultSvc = "external-dns.kube-system"

type lookupFunc func(indexKey string) ([]net.IP, endpoint.TTL)

type resourceWithIndex struct {
	name   string
	lookup lookupFunc
}

var orderedResources = []*resourceWithIndex{
	{
		name: "DNSEndpoint",
	},
}

var (
	ttlLowDefault     = uint32(60)
	ttlHighDefault    = uint32(3600)
	defaultApex       = "dns"
	defaultHostmaster = "hostmaster"
)

// Gateway stores all runtime configuration of a plugin
type Gateway struct {
	Next             plugin.Handler
	Zones            []string
	Resources        []*resourceWithIndex
	ttlLow           uint32
	ttlHigh          uint32
	Controller       *KubeController
	apex             string
	hostmaster       string
	Filter           string
	Annotation       string
	ExternalAddrFunc func(request.Request) []dns.RR
}

func newGateway() *Gateway {
	return &Gateway{
		apex:       defaultApex,
		Resources:  orderedResources,
		ttlLow:     ttlLowDefault,
		ttlHigh:    ttlHighDefault,
		hostmaster: defaultHostmaster,
	}
}

func lookupResource(resource string) *resourceWithIndex {

	for _, r := range orderedResources {
		if r.name == resource {
			return r
		}
	}
	return nil
}

func (gw *Gateway) updateResources(newResources []string) {

	gw.Resources = []*resourceWithIndex{}

	for _, name := range newResources {
		if resource := lookupResource(name); resource != nil {
			gw.Resources = append(gw.Resources, resource)
		}
	}
}

// ServeDNS implements the plugin.Handle interface.
func (gw *Gateway) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	log.Infof("Incoming query %s", state.QName())

	qname := state.QName()
	zone := plugin.Zones(gw.Zones).Matches(qname)
	if zone == "" {
		log.Infof("Request %s has not matched any zones %v", qname, gw.Zones)
		return plugin.NextOrFailure(gw.Name(), gw.Next, ctx, w, r)
	}
	zone = qname[len(qname)-len(zone):] // maintain case of original query
	state.Zone = zone

	// Computing keys to look up in cache
	indexKey := stripClosingDot(state.QName())

	log.Infof("Computed Index Keys %v", indexKey)

	if !gw.Controller.HasSynced() {
		// TODO maybe there's a better way to do this? e.g. return an error back to the client?
		return dns.RcodeServerFailure, plugin.Error(thisPlugin, fmt.Errorf("Could not sync required resources"))
	}

	for _, z := range gw.Zones {
		if state.Name() == z { // apex query
			ret, err := gw.serveApex(state)
			return ret, err
		}
		if dns.IsSubDomain(gw.apex+"."+z, state.Name()) {
			// dns subdomain test for ns. and dns. queries
			ret, err := gw.serveSubApex(state)
			return ret, err
		}
	}

	var addrs []net.IP
	var ttl endpoint.TTL

	// Iterate over supported resources and lookup DNS queries
	// Stop once we've found at least one match
	for _, resource := range gw.Resources {
		addrs, ttl = resource.lookup(indexKey)
		if len(addrs) > 0 {
			break
		}
	}
	log.Debugf("Computed response addresses %v", addrs)

	m := new(dns.Msg)
	m.SetReply(state.Req)

	if len(addrs) == 0 {
		m.Rcode = dns.RcodeNameError
		m.Ns = []dns.RR{gw.soa(state)}
		if err := w.WriteMsg(m); err != nil {
			log.Errorf("Failed to send a response: %s", err)
		}
		return 0, nil
	}

	switch state.QType() {
	case dns.TypeA:
		m.Answer = gw.A(state, addrs, ttl)
	default:
		m.Ns = []dns.RR{gw.soa(state)}
	}

	if len(m.Answer) == 0 {
		m.Ns = []dns.RR{gw.soa(state)}
	}

	if err := w.WriteMsg(m); err != nil {
		log.Errorf("Failed to send a response: %s", err)
	}

	return dns.RcodeSuccess, nil
}

// Name implements the Handler interface.
func (gw *Gateway) Name() string { return thisPlugin }

// A does the A-record lookup in ingress indexer
func (gw *Gateway) A(state request.Request, results []net.IP, ttl endpoint.TTL) (records []dns.RR) {
	dup := make(map[string]struct{})
	if !ttl.IsConfigured() {
		ttl = endpoint.TTL(gw.ttlLow)
	}
	for _, result := range results {
		if _, ok := dup[result.String()]; !ok {
			dup[result.String()] = struct{}{}
			records = append(records, &dns.A{Hdr: dns.RR_Header{Name: state.Name(), Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: uint32(ttl)}, A: result})
		}
	}
	return records
}

func (gw *Gateway) SelfAddress(state request.Request) (records []dns.RR) {
	// TODO: need to do self-index lookup for that i need
	// a) my own namespace - easy
	// b) my own serviceName - CoreDNS/k does that via localIP->Endpoint->Service
	// I don't really want to list Endpoints just for that so will fix that later
	//
	// As a workaround I'm reading an env variable (with a default)
	index := os.Getenv("EXTERNAL_SVC")
	if index == "" {
		index = defaultSvc
	}

	var addrs []net.IP
	var ttl endpoint.TTL
	for _, resource := range gw.Resources {
		addrs, ttl = resource.lookup(index)
		if len(addrs) > 0 {
			break
		}
	}

	m := new(dns.Msg)
	m.SetReply(state.Req)
	return gw.A(state, addrs, ttl)
}

// Strips the closing dot unless it's "."
func stripClosingDot(s string) string {
	if len(s) > 1 {
		return strings.TrimSuffix(s, ".")
	}
	return s
}
