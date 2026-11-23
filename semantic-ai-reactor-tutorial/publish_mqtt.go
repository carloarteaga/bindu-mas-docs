package main

import (
	cryptoTls "crypto/tls"
	"flag"
	"fmt"
	"os"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func main() {
	host := flag.String("host", "127.0.0.1", "MQTT Broker Host")
	port := flag.Int("port", 1883, "MQTT Broker Port")
	user := flag.String("user", "", "Username")
	pass := flag.String("pass", "", "Password")
	topic := flag.String("topic", "bindu/in", "Topic to publish to")
	msg := flag.String("msg", "", "JSON Message payload")
	flag.Parse()

	if *msg == "" {
		fmt.Fprintln(os.Stderr, "Error: -msg parameter is required")
		os.Exit(1)
	}

	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("ssl://%s:%d", *host, *port))
	tlsConfig := &cryptoTls.Config{
		InsecureSkipVerify: true,
	}
	opts.SetTLSConfig(tlsConfig)
	if *user != "" {
		opts.SetUsername(*user)
		opts.SetPassword(*pass)
	}
	opts.SetClientID(fmt.Sprintf("bindu-mqtt-pub-%d", time.Now().UnixNano()))
	opts.SetConnectTimeout(5 * time.Second)

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to MQTT broker tcp://%s:%d: %v\n", *host, *port, token.Error())
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
