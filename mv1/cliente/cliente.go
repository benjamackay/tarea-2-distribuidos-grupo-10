// mv1/cliente/cliente.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"

	monPB "tarea-2-distribuidos-grupo-10/proto/monitoreo"
	resPB "tarea-2-distribuidos-grupo-10/proto/reserva"

	"google.golang.org/grpc"
)

// ----- servidor para recibir notificaciones -----
type clienteMonitoreoServer struct {
	monPB.UnimplementedMonitoreoServiceServer
}

func (s *clienteMonitoreoServer) ActualizarCliente(ctx context.Context, est *monPB.EstadoReserva) (*monPB.ConfirmacionCliente, error) {
	log.Println("📥 actualización recibida:", est.GetMessage())
	return &monPB.ConfirmacionCliente{Received: true}, nil
}

func startMonitoreoServer() {
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatal(err)
	}
	srv := grpc.NewServer()
	monPB.RegisterMonitoreoServiceServer(srv, &clienteMonitoreoServer{})
	log.Println("✅ Cliente (MV1) escuchando monitoreo en :50053")
	go srv.Serve(lis)
}

type ReservaJSON struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	PartySize   int32  `json:"party_size"`
	Preferences string `json:"preferences"`
}

func main() {
	// 1. server de monitoreo
	startMonitoreoServer()

	// 2. leer reservas.json
	data, err := ioutil.ReadFile("reservas.json")
	if err != nil {
		log.Fatalf("no pude leer reservas.json: %v", err)
	}
	var reservas []ReservaJSON
	if err := json.Unmarshal(data, &reservas); err != nil {
		log.Fatalf("json inválido: %v", err)
	}
	log.Printf("📄 se cargaron %d reservas desde JSON\n", len(reservas))

	// 3. conectar al servicio de reservas en MV2
	// en PC: "localhost:50051"
	// en las VMs: "10.10.31.8:50051"
	addr := os.Getenv("RESERVA_ADDR")
	if addr == "" {
		addr = "localhost:50051"
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	cli := resPB.NewReservaServiceClient(conn)

	time.Sleep(1 * time.Second)

	// 4. armar ListaSolicitudes
	lista := &resPB.ListaSolicitudes{}
	for _, r := range reservas {
		lista.Solicitudes = append(lista.Solicitudes, &resPB.SolicitudReserva{
			Name:        r.Name,
			Phone:       r.Phone,
			PartySize:   r.PartySize,
			Preferences: r.Preferences,
		})
	}

	// 5. enviar TODAS juntas
	resp, err := cli.ProcesarReservas(context.Background(), lista)
	if err != nil {
		log.Fatalf("error llamando a ProcesarReservas: %v", err)
	}

	fmt.Println("respuesta MV2:", resp.GetMessage())

	// 6. quedarnos escuchando notificaciones de monitoreo
	select {}
}
