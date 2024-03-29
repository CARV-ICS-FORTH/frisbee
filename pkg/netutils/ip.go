/*
Copyright 2021-2023 ICS-FORTH.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package netutils

import (
	"io"
	"log"
	"net"
	"net/http"

	"github.com/pkg/errors"
)

// GetPublicIP asks a public IP API to return our public IP.
func GetPublicIP() (net.IP, error) {
	url := "https://api.ipify.org?format=text"
	// https://www.ipify.org
	// http://myexternalip.com
	// http://api.ident.me
	// http://whatismyipaddress.com/api
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "cannot contact public IP address API")
	}

	ipStr, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ip decoding error")
	}

	if err := resp.Body.Close(); err != nil {
		return nil, errors.Wrapf(err, "cannot close body")
	}

	return net.ParseIP(string(ipStr)), nil
}

// GetOutboundIP returns the preferred outbound ip of this machine
func GetOutboundIP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}
