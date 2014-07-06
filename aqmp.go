package main

import (
	"fmt"
	"github.com/streadway/amqp"
)


func publishMsg(cfg *Configuration, connection *amqp.Connection, msg string) error {
	// Get a Channel
	channel, err := connection.Channel()
	if err != nil {
		return fmt.Errorf("Channel: %s", err)
	}

	// Declare the Exchange
	if (cfg.Debug) {
		fmt.Println("Declaring Exchange ", cfg.Aqmp.Exchange)
	}

	if err := channel.ExchangeDeclare(
		cfg.Aqmp.Exchange,     // name
		cfg.Aqmp.ExchangeType, // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // noWait
		nil,          // arguments
	); err != nil {
		return fmt.Errorf("Exchange Declare: %s", err)
	}

/*
	if (cfg.Debug) {
		fmt.Println("Enable publishing confirm")
	}
	if err := channel.Confirm(false); err != nil {
		return fmt.Errorf("Channel could not be put into confirm mode: %s", err)
	}

	ack, nack := channel.NotifyConfirm(make(chan uint64, 1), make(chan uint64, 1))
	defer confirmOne(ack, nack)
*/

	// Send the Message
	if err = channel.Publish(
		cfg.Aqmp.Exchange,     // publish to an exchange
		cfg.Aqmp.RoutingKey, // routing to 0 or more queues
		false,      // mandatory
		false,      // immediate
		amqp.Publishing{
			Headers:         amqp.Table{},
			ContentType:     "text/plain",
			ContentEncoding: "",
			Body:            []byte(msg),
			DeliveryMode:    amqp.Transient, // 1=non-persistent, 2=persistent
			Priority:        0,              // 0-9
		},
	); err != nil {
		return fmt.Errorf("Exchange Publish: %s", err)
	}

	return nil
}

func cleanupConnection(cfg *Configuration, workNum int, connection *amqp.Connection) {
	if (cfg.Debug) {
		fmt.Println(fmt.Sprintf("[Worker %d] Closing connection", workNum))
	}
	connection.Close()
}

func AqmpWorker (cfg *Configuration, i int, aqmpStatus chan int, aqmpMessages chan string) {
	if cfg.Debug {
		fmt.Println("Initializing AQMP Worker", i)
	}

	// Set up Worker connections
	// "amqp://guest:guest@localhost:5672/"
	uri := fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
			cfg.Aqmp.User,
			cfg.Aqmp.Passwd,
			cfg.Aqmp.Host,
			cfg.Aqmp.Port,
			cfg.Aqmp.Vhost)

	if cfg.Debug {
		fmt.Printf(fmt.Sprintf("[Worker %d] Connecting to %q", i, uri))
	}

	// XXX Move this in a seperate function to be called
	// On reconnection as well
	connection, err := amqp.Dial(uri)
	if err != nil {
		fmt.Println(fmt.Errorf("[Worker %d] Connection error: %s", i, err))
		aqmpStatus <- -1
	}
	defer cleanupConnection(cfg, i, connection)

	// Positive value means success
	// TODO: Use an enum to allow for different states
	aqmpStatus <- 1

	// Listen for new incoming messages
	for {
		message := <-aqmpMessages
		publishMsg(cfg, connection, message)
		if cfg.Debug {
			fmt.Println(fmt.Sprintf("[Worker %d] Got message \"%s\"", i, message))
		}
	}
}
