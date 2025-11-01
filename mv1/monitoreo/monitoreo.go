package main

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"

	"tarea-2-distribuidos-grupo-10/proto/monitoreo"
)

func dialCliente(addr string) monitoreo.MonitoreoServiceClient {
	for {
		conn, err := grpc.Dial(addr, grpc.WithInsecure())
		if err == nil {
			return monitoreo.NewMonitoreoServiceClient(conn)
		}
		log.Println("cliente gRPC no disponible, reintentando en 2s:", err)
		time.Sleep(2 * time.Second)
	}
}

func main() {
	rmqConn, err := amqp.Dial("amqp://guest:guest@10.10.31.8:5672/")
	if err != nil {
		log.Fatal("no pude conectar a rabbitmq:", err)
	}
	defer rmqConn.Close()

	ch, err := rmqConn.Channel()
	if err != nil {
		log.Fatal("no pude abrir canal:", err)
	}
	defer ch.Close()

	// cola durable para no perder mensajes si el monitoreo parte antes
	_, err = ch.QueueDeclare(
		"reservas",
		true,  // durable
		false, // auto-delete
		false,
		false,
		nil,
	)
	if err != nil {
		log.Fatal("no pude declarar cola reservas:", err)
	}

	// NO la purgamos

	monCli := dialCliente("10.10.31.9:50053")

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
			log.Println("error avisando al cliente, reintentando conexión:", err)
			monCli = dialCliente("10.10.31.9:50053")
		}
	}
}
