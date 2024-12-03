package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {
	rabbitMQConnString := "amqp://guest:guest@localhost:5672/"
	conn, err := amqp.Dial(rabbitMQConnString)
	if err != nil {
		log.Fatalf("could not connect to the AMQP network %v", err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Println("connection could not close")
		}
	}()

	fmt.Println("Peril game server connected to RabbitMQ!")

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}
