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

	fmt.Println("Peril game client connected to RabbitMQ!")

	username, err := gamelogic.ClientWelcome()
	if err != nil {
		log.Fatalf("game logic failure: %v", err)
	}

	gameState := gamelogic.NewGameState(username)

	movePubCh, err := conn.Channel()
	if err != nil {
		log.Fatalf("cannot create channel: %v", err)
	}

	moveName := fmt.Sprintf("%s.%s", routing.ArmyMovesPrefix, username)
	moveKey := fmt.Sprintf("%s.*", routing.ArmyMovesPrefix)
	_, moveQueue, errMoveBind := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilTopic,
		moveName,
		moveKey,
		false,
	)
	if errMoveBind != nil {
		log.Fatalf("could not band move queue: %v", errMoveBind)
	}
	errMoveSubscribe := pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		moveQueue.Name,
		moveKey,
		false,
		handlerMove(gameState),
	)
	if errMoveSubscribe != nil {
		log.Fatalf("could not subscribe to move queue: %v", errMoveSubscribe)
	}

	pauseName := fmt.Sprintf("%s.%s", routing.PauseKey, username)
	_, _, errPauseBind := pubsub.DeclareAndBind(
		conn,
		routing.ExchangePerilDirect,
		pauseName,
		routing.PauseKey,
		false,
	)
	if errPauseBind != nil {
		log.Fatalf("could bind to pause queue: %v", errPauseBind)
	}
	errPauseSubscribe := pubsub.SubscribeJSON(
		conn,
		routing.ExchangePerilTopic,
		pauseName,
		fmt.Sprintf("%s.%s", routing.PauseKey, username),
		false,
		handlerPause(gameState),
	)
	if errPauseSubscribe != nil {
		log.Fatalf("could not subscribe to pause queue: %+v", err)
	}

OUTER:
	for {
		words := gamelogic.GetInput()
		if len(words) == 0 {
			continue
		}
		switch words[0] {
		case "spawn":
			err := gameState.CommandSpawn(words)
			if err != nil {
				log.Printf("error: %v", err)
			}

		case "move":
			armyMove, err := gameState.CommandMove(words)
			if err != nil {
				log.Printf("error: %v", err)
				continue OUTER
			}
			pubsub.PublishJSON(
				movePubCh,
				routing.ExchangePerilTopic,
				moveKey,
				&armyMove,
			)

			fmt.Println("Move successful!")

		case "status":
			gameState.CommandStatus()

		case "help":
			gamelogic.PrintClientHelp()

		case "spam":
			fmt.Println("Spamming not allowed yet")

		case "quit":
			fmt.Println("Quitting!")
			break OUTER

		default:
			fmt.Println("No such command.")
		}
	}
}
