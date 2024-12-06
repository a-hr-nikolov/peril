package main

import (
	"fmt"
	"log"

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

	fmt.Println("Peril game server connected to RabbitMQ!")

	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("could not create rabbitMQ channel: %v", err)
	}
	defer func() {
		if err := channel.Close(); err != nil {
			log.Println("could not close:", err)
		}
	}()

	gamelogic.PrintServerHelp()

	err = pubsub.SubscribeGob(
		conn,
		routing.ExchangePerilTopic,
		routing.GameLogSlug,
		routing.GameLogSlug+".*",
		true,
		handlerLog(),
	)
	if err != nil {
		log.Fatalf("could not subscribe to logging queue: %v", err)
	}

OUTER:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "pause":
			fmt.Println("Sending a pause message to RabbitMQ!")
			pubsub.PublishJSON(
				channel,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				&routing.PlayingState{IsPaused: true},
			)
		case "resume":
			fmt.Println("Sending a resume message to RabbitMQ!")
			pubsub.PublishJSON(
				channel,
				routing.ExchangePerilDirect,
				routing.PauseKey,
				routing.PlayingState{IsPaused: false},
			)
		case "quit":
			fmt.Println("Quitting")
			break OUTER
		default:
			fmt.Println("No such command.")
		}
	}
}
