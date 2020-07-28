package main

import (
	"crypto/rand"

	"golang.org/x/crypto/curve25519"
)

const keySize = 32

var random = rand.Read

func GeneratePrivateKey() ([32]byte, error) {
	var b [32]byte
	_, err := random(b[:])
	return b, err
}

func GeneratePublicKey(priv [32]byte) ([32]byte, error) {
	var pub [32]byte
	curve25519.ScalarBaseMult(&pub, &priv)
	return pub, nil
}
