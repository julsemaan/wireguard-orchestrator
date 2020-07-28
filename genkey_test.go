package main

import (
	"encoding/base64"
	"math/rand"
	"testing"

	"github.com/inverse-inc/packetfence/go/sharedutils"
)

func TestGenkey(t *testing.T) {
	random = rand.Read

	priv, err := GeneratePrivateKey()
	sharedutils.CheckError(err)
	priv64 := base64.StdEncoding.EncodeToString(priv[:])
	pub, err := GeneratePublicKey(priv)
	sharedutils.CheckError(err)
	pub64 := base64.StdEncoding.EncodeToString(pub[:])

	if priv64 != "Uv38ByGCZU8WP18PmmIdcpVmx00QA3xNe7sEB9Hixkk=" {
		t.Error("Unexpected value for private key")
	}
	if pub64 != "ZP/Mzlvt9BwNH9oqtuL0ZP8OW1foBBWfE8R6nSrM/nk=" {
		t.Error("Unexpected value for public key")
	}
}
