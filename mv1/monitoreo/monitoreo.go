package main

import (
	"context"
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"

	"tarea-2-distribuidos-grupo-10/proto/monitoreo"
)

func main() {
	// RabbitMQ (cambiar a 10.10.31.7 si es la VM)
	rmqConn, err := amqp.Dial("amqp://guest:guest@10.10.31.7:5672/")
	if err != nil {
		log.Fatal("no pude conectar a rabbitmq:", err)
	}
	defer rmqConn.Close()

	ch, err := rmqConn.Channel()
	if err != nil {
		log.Fatal("no pude abrir canal:", err)
	}
	defer ch.Close()

	_, err = ch.QueueDeclare("reservas", false, false, false, false, nil)
	if err != nil {
		log.Fatal("no pude declarar cola reservas:", err)
	}

	_, _ = ch.QueuePurge("reservas", false)

	// cambiar a "10.10.31.7:50053" si el cliente está en la VM
	cliConn, err := grpc.Dial("10.10.31.7:50053", grpc.WithInsecure())
	if err != nil {
		log.Fatal("no pude conectar al cliente:", err)
	}
	defer cliConn.Close()
	monCli := monitoreo.NewMonitoreoServiceClient(cliConn)

	msgs, err := ch.Consume("reservas", "", true, false, false, false, nil)
	if err != nil {
		log.Fatal("no pude leer cola reservas:", err)
	}

	log.Println("Monitoreo (MV1) escuchando cola 'reservas' y reenviando al cliente...")

	for m := range msgs {
		body := string(m.Body)
		log.Println("mensaje de cola:", body)

		_, err := monCli.ActualizarCliente(context.Background(), &monitoreo.EstadoReserva{
			Name:    "sistema",
			Status:  "actualizacion",
			Message: body,
		})
		if err != nil {
			log.Println("error avisando al cliente:", err)
		}
	}
}
