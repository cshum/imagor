package config

import (
	"net"
	"strings"
)

// CIDRSliceFlag is a flag type which support comma separated CIDR expressions.
type CIDRSliceFlag []*net.IPNet

// String implements flag.Setter interface
func (s *CIDRSliceFlag) String() string {
	var ss []string
	for _, v := range *s {
		ss = append(ss, v.String())
	}
	return strings.Join(ss, ",")
}

// Set implements flag.Setter interface
func (s *CIDRSliceFlag) Set(value string) error {
	var res []*net.IPNet
	for _, v := range strings.Split(value, ",") {
		_, network, err := net.ParseCIDR(v)
		if err != nil {
			return err
		}
		res = append(res, network)
	}
	*s = res
	return nil
}

// Get implements flag.Getter interface
func (s *CIDRSliceFlag) Get() any {
	return s
}
