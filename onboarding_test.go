package main

import (
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	httpexpect "github.com/gavv/httpexpect/v2"
	"github.com/gin-gonic/gin"
)

var testServer *httptest.Server

var peerEvents = struct {
	sync.Mutex
	data map[string]Event
}{data: map[string]Event{}}

func init() {
	router := makeGinServer()
	testServer = httptest.NewServer(router)
}

func TestOnboarding(t *testing.T) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go onboard(t, wg, "peer1")
	wg.Add(1)
	go onboard(t, wg, "peer2")
	wg.Add(1)
	go onboard(t, wg, "peer3")
	wg.Add(1)
	go onboard(t, wg, "peer4")
	wg.Wait()
}

func onboard(t *testing.T, wg *sync.WaitGroup, myID string) {
	e := httpexpect.New(t, testServer.URL)

	profile := peers[myID]

	e.GET("/profile/"+myID).
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		ValueEqual("private_key", profile.PrivateKey).
		ValueEqual("public_key", profile.PublicKey).
		ValueEqual("wireguard_ip", profile.WireguardIP).
		ValueEqual("wireguard_netmask", profile.WireguardNetmask).
		ValueEqual("allowed_peers", profile.AllowedPeers)

	wg2 := &sync.WaitGroup{}
	for _, peerID := range profile.AllowedPeers {
		wg2.Add(1)
		go func(peerID string) {
			peerProfile := peers[peerID]

			peerTest := e.GET("/peer/" + peerID).
				Expect().
				Status(http.StatusOK).
				JSON().Object()

			peerTest.Keys().ContainsOnly("wireguard_ip", "public_key")

			peerTest.
				ValueEqual("wireguard_ip", peerProfile.WireguardIP).
				ValueEqual("public_key", peerProfile.PublicKey)

			p2pk := buildP2PKey(profile.PublicKey, peerProfile.PublicKey)

			ip := net.IPv4(byte(rand.Intn(255)), byte(rand.Intn(255)), byte(rand.Intn(255)), byte(rand.Intn(255)))
			port := rand.Intn(65536)

			data := Event{
				Type: "public_endpoint",
				Data: gin.H{
					"id":              profile.PublicKey,
					"public_endpoint": fmt.Sprintf("%s:%d", ip, port),
				},
			}

			e.POST("/events/" + p2pk).WithJSON(data).Expect().
				Status(http.StatusOK)

			peerEvents.Lock()
			peerEvents.data[p2pk+myID] = data
			peerEvents.Unlock()

			pollPeerData(t, e, profile, peerProfile, p2pk, data)

			for {
				peerEvents.Lock()
				if peerData, ok := peerEvents.data[p2pk+peerID]; ok {
					peerEvents.Unlock()
					pollPeerData(t, e, profile, peerProfile, p2pk, peerData)
					break
				} else {
					peerEvents.Unlock()
				}
			}
			wg2.Done()
		}(peerID)
	}

	wg2.Wait()
	wg.Done()
}

func pollPeerData(t *testing.T, e *httpexpect.Expect, profile, peerProfile Peer, p2pk string, data Event) {
	events := e.GET("/events/"+p2pk).WithQuery("since_time", "0").WithQuery("timeout", "1").
		Expect().
		Status(http.StatusOK).
		JSON().Object().
		Value("events").
		Array().Raw()

	foundData := false
	for _, o := range events {
		e := o.(map[string]interface{})
		serverData := e["data"].(map[string]interface{})["data"].(map[string]interface{})
		if serverData["id"] == data.Data["id"] && serverData["public_endpoint"] == data.Data["public_endpoint"] {
			foundData = true
		}
	}

	if !foundData {
		t.Error("Was unable to find check-in data in the events for ID", data.Data["id"], ", public endpoint", data.Data["public_endpoint"])
	}
}
