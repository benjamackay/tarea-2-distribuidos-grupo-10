package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	monitoreoPB "tarea-2-distribuidos-grupo-10/proto/monitoreo"
	reservaPB "tarea-2-distribuidos-grupo-10/proto/reserva"
)

type Reserva struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	PartySize   int32  `json:"party_size"`
	Preferences string `json:"preferences"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Uso: go run mv1/cliente/cliente.go <archivo.json>")
	}
	jsonFile := os.Args[1]

	data, err := os.ReadFile(jsonFile)
	if err != nil {
		log.Fatalf("no pude leer %s: %v", jsonFile, err)
	}

	var reservas []Reserva
	if err := json.Unmarshal(data, &reservas); err != nil {
		log.Fatalf("json inválido: %v", err)
	}

	go iniciarServidorMonitoreo() // escucha en :50053

	// MV2 (reservas) -> cambiar localhost a 10.10.31.8 en las VMs
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatal("no pude conectar a MV2:", err)
	}
	defer conn.Close()
	resCli := reservaPB.NewReservaServiceClient(conn)

	lista := &reservaPB.ListaSolicitudes{}
	for _, r := range reservas {
		lista.Solicitudes = append(lista.Solicitudes, &reservaPB.SolicitudReserva{
			Name:        r.Name,
			Phone:       r.Phone,
			PartySize:   r.PartySize,
			Preferences: r.Preferences,
		})
	}

	resp, err := resCli.ProcesarReservas(context.Background(), lista)
	if err != nil {
		log.Fatal("error llamando a ProcesarReservas:", err)
	}

	fmt.Println(resp.Message)
}

func iniciarServidorMonitoreo() {
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatal("no pude escuchar monitoreo:", err)
	}
	s := grpc.NewServer()
	monitoreoPB.RegisterMonitoreoServiceServer(s, &monitoreoServer{})
	log.Println("Cliente (MV1) escuchando monitoreo en :50053")
	if err := s.Serve(lis); err != nil {
		log.Fatal("error sirviendo monitoreo:", err)
	}
}

type monitoreoServer struct {
	monitoreoPB.UnimplementedMonitoreoServiceServer
}

func (m *monitoreoServer) ActualizarCliente(ctx context.Context, estado *monitoreoPB.EstadoReserva) (*monitoreoPB.ConfirmacionCliente, error) {
	log.Printf("📥 actualización recibida: %s", estado.Message)
	return &monitoreoPB.ConfirmacionCliente{Received: true}, nil
}
