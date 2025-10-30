// mv1/monitoreo/monitoreo.go
package main

import (
	"context"
	"log"

	"tarea-2-distribuidos-grupo-10/proto/monitoreo"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
)

func main() {
	// 1. conectar a RabbitMQ
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		log.Fatal("no pude conectar a rabbitmq:", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatal("no pude abrir canal:", err)
	}
	defer ch.Close()

	msgs, err := ch.Consume("reservas", "", true, false, false, false, nil)
	if err != nil {
		log.Fatal("no pude leer cola reservas:", err)
	}

	// 2. conectar al cliente (que escucha monitoreo en 50053)
	cliConn, err := grpc.Dial("localhost:50053", grpc.WithInsecure())
	if err != nil {
		log.Fatal("no pude conectar al cliente:", err)
	}
	defer cliConn.Close()
	monCli := monitoreo.NewMonitoreoServiceClient(cliConn)

	log.Println("🟣 Monitoreo (MV1) escuchando cola 'reservas' y reenviando al cliente...")
	for m := range msgs {
		body := string(m.Body)
		log.Println("📦 mensaje de cola:", body)

		_, err := monCli.ActualizarCliente(context.Background(), &monitoreo.EstadoReserva{
			Name:    "sistema",
			Status:  "actualizacion",
			Message: body,
		})
		if err != nil {
			log.Println("⚠️ error avisando al cliente:", err)
		}
	}
}
