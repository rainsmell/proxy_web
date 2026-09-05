package main

import (
	"encoding/json"
	"testing"
)

func TestParseVLESS(t *testing.T) {
	n, err := parseVLESS("vless://11111111-1111-1111-1111-111111111111@example.com:443?type=ws&security=tls&sni=cdn.example.com&host=cdn.example.com&path=%2Fws#demo")
	if err != nil {
		t.Fatal(err)
	}
	if n.Protocol != "vless" || n.Port != 443 || n.Network != "ws" || !n.TLS || n.Path != "/ws" {
		t.Fatalf("unexpected node: %+v", n)
	}
}

func TestVLESSOutboundEncryption(t *testing.T) {
	o := buildOutbound(Node{Protocol: "vless", Address: "example.com", Port: 443, UUID: "11111111-1111-1111-1111-111111111111"})
	settings := o["settings"].(map[string]any)
	vnext := settings["vnext"].([]any)
	users := vnext[0].(map[string]any)["users"].([]any)
	if users[0].(map[string]any)["encryption"] != "none" {
		t.Fatal("VLESS encryption must be none")
	}
}

func TestStateJSONIncludesNodeName(t *testing.T) {
	b, err := json.Marshal(State{Running: true, NodeID: "b303d4bf9751ab8e", NodeName: "日本东京09", Error: "sample"})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["nodeName"] != "日本东京09" {
		t.Fatalf("nodeName missing or wrong: %s", b)
	}
	if got["error"] != "sample" {
		t.Fatalf("error missing or wrong: %s", b)
	}
}
