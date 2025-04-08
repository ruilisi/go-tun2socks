package tun

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os/exec"
	"strconv"
	"strings"

	"github.com/songgao/water"
)

func isIPv4(ip net.IP) bool {
	if ip.To4() != nil {
		return true
	}
	return false
}

func isIPv6(ip net.IP) bool {
	// To16() also valid for ipv4, ensure it's not an ipv4 address
	if ip.To4() != nil {
		return false
	}
	if ip.To16() != nil {
		return true
	}
	return false
}

func randomIPv6LinkLocalAddr() string {
	size := 16
	ip := make([]byte, size)
	ip[0] = 0xfe
	ip[1] = 0x80
	for i := 8; i < size; i++ {
		ip[i] = byte(rand.Intn(256))
	}
	return net.IP(ip).To16().String()
}

func OpenTunDevice(name, addr, gw, mask string, dns []string, persist bool) (io.ReadWriteCloser, error) {
	genErr := func(out []byte, err error) error {
		if err == nil {
			return nil
		}
		if len(out) != 0 {
			return errors.New(fmt.Sprintf("%v, output: %s", err, out))
		}
		return err
	}
	tunDev, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		return nil, err
	}
	name = tunDev.Name()
	ip := net.ParseIP(addr)
	if ip == nil {
		return nil, errors.New("invalid IP address")
	}

	var params string
	if isIPv4(ip) {
		params = fmt.Sprintf("%s inet %s netmask %s %s", name, addr, mask, gw)
		out, err := exec.Command("ifconfig", strings.Split(params, " ")...).Output()
		if err != nil {
			return nil, genErr(out, err)
		}
		params = fmt.Sprintf("%s inet6 %s/64", name, randomIPv6LinkLocalAddr())
		out, err = exec.Command("ifconfig", strings.Split(params, " ")...).Output()
		if err != nil {
			return nil, genErr(out, err)
		}
	} else if isIPv6(ip) {
		prefixlen, err := strconv.Atoi(mask)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("parse IPv6 prefixlen failed: %v", err))
		}
		params = fmt.Sprintf("%s inet6 %s/%d", name, addr, prefixlen)
		out, err := exec.Command("ifconfig", strings.Split(params, " ")...).Output()
		if err != nil {
			return nil, genErr(out, err)
		}
	} else {
		return nil, errors.New("invalid IP address")
	}

	return tunDev, nil
}
