package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	host := flag.String("host", "127.0.0.1", "MQTT Broker Host")
	port := flag.Int("port", 1887, "MQTT Broker Port")
	user := flag.String("user", "console", "Username")
	pass := flag.String("pass", "StrongPass123", "Password")
	useTLS := flag.Bool("tls", true, "Use TLS/SSL")
	topic := flag.String("topic", "bindu/in", "Topic to publish to")
	msg := flag.String("msg", "", "JSON Message payload")
	flag.Parse()

	if *msg == "" {
		fmt.Fprintln(os.Stderr, "Error: -msg parameter is required")
		os.Exit(1)
	}

	scheme := "ssl"
	if !*useTLS && strings.HasPrefix(*host, "tcp://") {
		scheme = "tcp"
	}
	cleanHost := strings.TrimPrefix(strings.TrimPrefix(strings.TrimPrefix(*host, "ssl://"), "tls://"), "tcp://")

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%d", scheme, cleanHost, *port))
	if scheme == "ssl" {
		opts.SetTLSConfig(&tls.Config{
			InsecureSkipVerify: true,
		})
	}
	if *user != "" {
		opts.SetUsername(*user)
		opts.SetPassword(*pass)
	}
	opts.SetClientID(fmt.Sprintf("bindu-rest-sim-pub-%d", time.Now().UnixNano()))
	opts.SetConnectTimeout(5 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to MQTT broker %s://%s:%d: %v\n", scheme, cleanHost, *port, token.Error())
		os.Exit(1)
	}
	defer client.Disconnect(250)

	token := client.Publish(*topic, 1, false, *msg)
	token.Wait()
	if token.Error() != nil {
		fmt.Fprintf(os.Stderr, "Error publishing to topic %s: %v\n", *topic, token.Error())
		os.Exit(1)
	}
}
