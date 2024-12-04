package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/bootdotdev/learn-pub-sub-starter/internal/gamelogic"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/pubsub"
	"github.com/bootdotdev/learn-pub-sub-starter/internal/routing"
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

	fmt.Println("Peril game client connected to RabbitMQ!")

	user, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	queueName := fmt.Sprintf("%s.%s", routing.PauseKey, user)
	_, _, errBind := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		queueName,
		routing.PauseKey,
		false,
	)
	if errBind != nil {
		log.Fatalf("Could not bind queue: %v", errBind)
	}

	// wait for ctrl+c
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt)
	<-signalChan
}
