package main

import (
	"encoding/base64"
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
	WireguardIP      net.IP   `json:"wireguard_ip"`
	WireguardNetmask int      `json:"wireguard_netmask"`
	PublicKey        string   `json:"public_key"`
	PrivateKey       string   `json:"private_key"`
	AllowedPeers     []string `json:"allowed_peers"`
}

var peers = map[string]*Peer{
	"peer1": &Peer{
		WireguardIP:      net.ParseIP("192.168.69.11"),
		WireguardNetmask: 24,
		AllowedPeers:     []string{"peer2", "peer3", "peer4"},
	},
	"peer2": &Peer{
		WireguardIP:      net.ParseIP("192.168.69.12"),
		WireguardNetmask: 24,
		AllowedPeers:     []string{"peer1", "peer4"},
	},
	"peer3": &Peer{
		WireguardIP:      net.ParseIP("192.168.69.13"),
		WireguardNetmask: 24,
		AllowedPeers:     []string{"peer1", "peer4"},
	},
	"peer4": &Peer{
		WireguardIP:      net.ParseIP("192.168.69.14"),
		WireguardNetmask: 24,
		AllowedPeers:     []string{"peer1", "peer2", "peer3"},
	},
}

func init() {
	for _, peer := range peers {
		priv, err := GeneratePrivateKey()
		sharedutils.CheckError(err)
		privslice := priv[:]
		peer.PrivateKey = base64.StdEncoding.EncodeToString(privslice)

		pub, err := GeneratePublicKey(priv)
		sharedutils.CheckError(err)
		pubslice := pub[:]
		peer.PublicKey = base64.StdEncoding.EncodeToString(pubslice)
	}
}

func handleGetProfile(c *gin.Context) {
	if peer, ok := peers[c.Param("node_id")]; ok {
		c.JSON(http.StatusOK, peer)
	} else {
		renderError(c, http.StatusNotFound, errors.New("Unable to find a peer with this identifier"))
	}
}

func handleGetPeer(c *gin.Context) {
	if peer, ok := peers[c.Param("node_id")]; ok {
		c.JSON(http.StatusOK, gin.H{"public_key": peer.PublicKey, "wireguard_ip": peer.WireguardIP})
	} else {
		renderError(c, http.StatusNotFound, errors.New("Unable to find a peer with this identifier"))
	}
}

func handleGetEvents(c *gin.Context) {
	if lp := longPollFromContext(c); lp != nil {
		lp.SubscriptionHandler(c.Writer, c.Request)
	} else {
		renderError(c, http.StatusInternalServerError, errors.New("Unable to find events manager in context"))
	}
}
func handleGetEventsKey(c *gin.Context) {
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
		c.Request.URL, err = url.Parse(c.Request.URL.String() + char + "category=" + k + timeout)
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

func buildP2PKey(key1, key2 string) string {
	if key2 < key1 {
		key1bak := key1
		key1 = key2
		key2 = key1bak
	}

	key1dec, err := base64.StdEncoding.DecodeString(key1)
	sharedutils.CheckError(err)
	key2dec, err := base64.StdEncoding.DecodeString(key2)
	sharedutils.CheckError(err)

	combined := append(key1dec, key2dec...)
	return base64.URLEncoding.EncodeToString(combined)
}
