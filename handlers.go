package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/inverse-inc/packetfence/go/sharedutils"
)

type Peer struct {
	WireguardIP net.IP `json:"wireguard_ip"`
	PublicKey   string `json:"public_key"`
	PrivateKey  string `json:"private_key"`
}

var peers = map[string]Peer{
	"peer1": Peer{
		WireguardIP: net.ParseIP("192.168.69.11"),
		PrivateKey:  "kJ4y53ahQJdhlEsJ7RGqXcnF1lQngrsCIeIR/n4gQUk=",
		PublicKey:   "N+ZrArw5fXck3iolfeVm85VhsfZD0TLkGH8Yqg/YImQ=",
	},
	"peer2": Peer{
		WireguardIP: net.ParseIP("192.168.69.12"),
		PrivateKey:  "ADbl0gotxpZVxa1XK9fmaN1maAI4BC3n2otJ5KENj1g=",
		PublicKey:   "yhgu58zhYbv+wzfTVbjMb+AZ3eZwEbBG2tHB7mtMfHc=",
	},
	"peer3": Peer{
		WireguardIP: net.ParseIP("192.168.69.13"),
		PrivateKey:  "0CptpDd2Mvd359CvIN3jVmE9LOB+nBVxob+i0zTbOGY=",
		PublicKey:   "4s6iiDZA5lfqXEIJe1CrcgWJfl4OzhiobGdg+RI7axc=",
	},
}

func handleGetProfile(c *gin.Context) {
	if peer, ok := peers[c.Param("node_id")]; ok {
		c.JSON(http.StatusOK, peer)
	} else {
		renderError(c, http.StatusNotFound, errors.New("Unable to find a peer with this identifier"))
	}
}

func handleGetEvents(c *gin.Context) {
	if lp := longPollFromContext(c); lp != nil {
		k := c.Param("k")

		char := "?"
		if strings.Contains(c.Request.URL.String(), "?") {
			char = "&"
		}

		timeout := ""
		if _, ok := c.GetQuery("timeout"); !ok {
			timeout = fmt.Sprintf("&timeout=%d", defaultPollTimeout/time.Second)
		}

		var err error
		c.Request.URL, err = url.Parse(c.Request.URL.String() + char + "category=" + k + "&since_time=0" + timeout)
		sharedutils.CheckError(err)

		lp.SubscriptionHandler(c.Writer, c.Request)
	} else {
		renderError(c, http.StatusInternalServerError, errors.New("Unable to find events manager in context"))
	}
}

type Event struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
}

func handlePostEvents(c *gin.Context) {
	if lp := longPollFromContext(c); lp != nil {
		e := Event{}
		if err := c.BindJSON(&e); err == nil {
			lp.Publish(c.Param("k"), e)
		} else {
			renderError(c, http.StatusBadRequest, errors.New("Unable to parse JSON payload: "+err.Error()))
		}
	} else {
		renderError(c, http.StatusInternalServerError, errors.New("Unable to find events manager in context"))
	}
}
