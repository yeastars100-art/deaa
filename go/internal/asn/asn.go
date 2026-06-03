package asn

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

type Prefix struct {
	Prefix string `json:"prefix"`
}

type BGPViewResponse struct {
	Status string `json:"status"`
	Data   struct {
		IPv4Prefixes []Prefix `json:"ipv4_prefixes"`
	} `json:"data"`
}

func FetchPrefixes(asn string) ([]string, error) {
	asn = strings.TrimPrefix(strings.ToUpper(asn), "AS")
	url := fmt.Sprintf("https://api.bgpview.io/asn/%s/prefixes", asn)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data BGPViewResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	var prefixes []string
	for _, p := range data.Data.IPv4Prefixes {
		prefixes = append(prefixes, p.Prefix)
	}
	return prefixes, nil
}

func ExpandCIDR(cidr string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, err
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
		if len(ips) > 65536 { // Safety limit for massive ranges
			break
		}
	}
	// Remove network and broadcast
	if len(ips) > 2 {
		return ips[1 : len(ips)-1], nil
	}
	return ips, nil
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}


