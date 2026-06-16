package gopubssh

import (
	"reflect"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestClientConfigModernUsesSupportedAlgorithms(t *testing.T) {
	config := (&remoteSession{algorithm: SSHAlgorithmModern}).clientConfig()
	supported := ssh.SupportedAlgorithms()

	if !reflect.DeepEqual(config.Config.KeyExchanges, supported.KeyExchanges) {
		t.Fatalf("unexpected modern key exchanges: %v", config.Config.KeyExchanges)
	}
	if !reflect.DeepEqual(config.Config.Ciphers, supported.Ciphers) {
		t.Fatalf("unexpected modern ciphers: %v", config.Config.Ciphers)
	}
	if !reflect.DeepEqual(config.Config.MACs, supported.MACs) {
		t.Fatalf("unexpected modern MACs: %v", config.Config.MACs)
	}
	if !reflect.DeepEqual(config.HostKeyAlgorithms, supported.HostKeys) {
		t.Fatalf("unexpected modern host keys: %v", config.HostKeyAlgorithms)
	}
}

func TestClientConfigLegacyAppendsInsecureAlgorithms(t *testing.T) {
	config := (&remoteSession{algorithm: SSHAlgorithmLegacy}).clientConfig()
	supported := ssh.SupportedAlgorithms()
	insecure := ssh.InsecureAlgorithms()

	wantKeyExchanges := appendUnique(supported.KeyExchanges, insecure.KeyExchanges...)
	wantCiphers := appendUnique(supported.Ciphers, insecure.Ciphers...)
	wantMACs := appendUnique(supported.MACs, insecure.MACs...)
	wantHostKeys := appendUnique(supported.HostKeys, insecure.HostKeys...)

	if !reflect.DeepEqual(config.Config.KeyExchanges, wantKeyExchanges) {
		t.Fatalf("unexpected legacy key exchanges: %v", config.Config.KeyExchanges)
	}
	if !reflect.DeepEqual(config.Config.Ciphers, wantCiphers) {
		t.Fatalf("unexpected legacy ciphers: %v", config.Config.Ciphers)
	}
	if !reflect.DeepEqual(config.Config.MACs, wantMACs) {
		t.Fatalf("unexpected legacy MACs: %v", config.Config.MACs)
	}
	if !reflect.DeepEqual(config.HostKeyAlgorithms, wantHostKeys) {
		t.Fatalf("unexpected legacy host keys: %v", config.HostKeyAlgorithms)
	}
}
