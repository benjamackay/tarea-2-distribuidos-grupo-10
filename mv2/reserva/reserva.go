package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"regexp"
	"sort"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"

	registroPB "tarea-2-distribuidos-grupo-10/proto/registro"
	reservaPB "tarea-2-distribuidos-grupo-10/proto/reserva"
)

type Mesa struct {
	TableID  string `bson:"table_id"`
	Capacity int    `bson:"capacity"`
	Status   string `bson:"status"`
	Tipo     string `bson:"tipo"`
}

type ReservaServer struct {
	reservaPB.UnimplementedReservaServiceServer
	dbMesas  *mongo.Collection
	registro registroPB.RegistroServiceClient
	rabbitCh *amqp.Channel
}

func (s *ReservaServer) EnviarSolicitud(ctx context.Context, req *reservaPB.SolicitudReserva) (*reservaPB.ConfirmacionRecepcion, error) {
	msg, ok := s.procesarUna(ctx, req)
	return &reservaPB.ConfirmacionRecepcion{Success: ok, Message: msg}, nil
}

func (s *ReservaServer) ProcesarReservas(ctx context.Context, lista *reservaPB.ListaSolicitudes) (*reservaPB.ConfirmacionRecepcion, error) {
	var b strings.Builder
	b.WriteString("Solicitudes de reserva recibidas\n")
	for _, r := range lista.Solicitudes {
		lineas, _ := s.procesarUna(ctx, r)
		if !strings.HasSuffix(lineas, "\n") {
			lineas += "\n"
		}
		b.WriteString(lineas)
	}
	return &reservaPB.ConfirmacionRecepcion{Success: true, Message: b.String()}, nil
}

func (s *ReservaServer) procesarUna(ctx context.Context, req *reservaPB.SolicitudReserva) (string, bool) {
	name := req.GetName()
	pax := req.GetPartySize()
	zone := normalizaZona(req.GetPreferences())

	fDisp := bson.M{"status": "Disponible"}
	cur, err := s.dbMesas.Find(ctx, fDisp)
	if err != nil {
		return fmt.Sprintf("Reserva de %s para %d personas en zona %s fallida.\nError de base de datos.",
			name, pax, zone), false
	}
	defer cur.Close(ctx)

	var todas []Mesa
	for cur.Next(ctx) {
		var m Mesa
		if err := cur.Decode(&m); err == nil {
			todas = append(todas, m)
		}
	}

	var matchZoneCap, capAnyZone, anyInZone []Mesa
	for _, m := range todas {
		if strings.EqualFold(normalizaZona(m.Tipo), zone) {
			anyInZone = append(anyInZone, m)
			if m.Capacity >= int(pax) {
				matchZoneCap = append(matchZoneCap, m)
			}
		}
		if m.Capacity >= int(pax) {
			capAnyZone = append(capAnyZone, m)
		}
	}

	sort.Slice(matchZoneCap, func(i, j int) bool { return matchZoneCap[i].Capacity < matchZoneCap[j].Capacity })
	sort.Slice(capAnyZone, func(i, j int) bool { return capAnyZone[i].Capacity < capAnyZone[j].Capacity })

	if len(matchZoneCap) > 0 {
		m := matchZoneCap[0]
		if s.reservaMesa(ctx, m.TableID) {
			s.notificarRegistro(ctx, req, m)
			s.publicarCola(true, name, m.TableID, zone, pax)
			idFmt := formateaID(m.TableID)
			return fmt.Sprintf("Reserva de %s para %d personas en zona %s exitosa.\nSe ha asignado %s (capacidad %d personas) en zona %s.",
				name, pax, zone, idFmt, m.Capacity, normalizaZona(m.Tipo)), true
		}
	}

	if len(capAnyZone) > 0 {
		m := capAnyZone[0]
		if s.reservaMesa(ctx, m.TableID) {
			s.notificarRegistro(ctx, req, m)
			s.publicarCola(true, name, m.TableID, zone, pax)
			idFmt := formateaID(m.TableID)
			return fmt.Sprintf("Reserva de %s para %d personas en zona %s exitosa con modificaciones.\nSe ha asignado %s (capacidad %d personas) en zona %s.",
				name, pax, zone, idFmt, m.Capacity, normalizaZona(m.Tipo)), true
		}
	}

	if len(anyInZone) > 0 {
		s.publicarCola(false, name, "", zone, pax)
		return fmt.Sprintf("Reserva de %s para %d personas en zona %s fallida.\nNo hay mesas con esa capacidad disponibles.",
			name, pax, zone), false
	}

	s.publicarCola(false, name, "", zone, pax)
	return fmt.Sprintf("Reserva de %s para %d personas en zona %s fallida.\nNo hay mesas en zona %s disponibles.",
		name, pax, zone, zone), false
}

func (s *ReservaServer) reservaMesa(ctx context.Context, tableID string) bool {
	_, err := s.dbMesas.UpdateOne(ctx, bson.M{"table_id": tableID, "status": "Disponible"}, bson.M{"$set": bson.M{"status": "Reservada"}})
	return err == nil
}

func (s *ReservaServer) notificarRegistro(ctx context.Context, req *reservaPB.SolicitudReserva, m Mesa) {
	if s.registro == nil {
		return
	}
	_, _ = s.registro.RegistrarReserva(ctx, &registroPB.ReservaConfirmada{
		ReservationId: fmt.Sprintf("res-%d", time.Now().UnixNano()),
		Name:          req.GetName(),
		Phone:         req.GetPhone(),
		PartySize:     req.GetPartySize(),
		Preferences:   req.GetPreferences(),
		TableId:       m.TableID,
		Status:        "Confirmada",
	})
}

func (s *ReservaServer) publicarCola(ok bool, name, tableID, pref string, pax int32) {
	if s.rabbitCh == nil {
		return
	}
	var body string
	if ok {
		body = fmt.Sprintf("OK|%s|%s|%s|%d", name, tableID, pref, pax)
	} else {
		body = fmt.Sprintf("FAIL|%s|%s|%d", name, pref, pax)
	}
	_ = s.rabbitCh.Publish("", "reservas", false, false, amqp.Publishing{ContentType: "text/plain", Body: []byte(body)})
}

func normalizaZona(z string) string {
	z = strings.TrimSpace(strings.ToLower(z))
	switch z {
	case "fumador", "fumadores":
		return "fumadores"
	case "no fumador", "no fumadores", "nofumadores", "no-fumadores":
		return "no fumadores"
	case "interior":
		return "interior"
	case "exterior":
		return "exterior"
	default:
		return z
	}
}

var reMesaNum = regexp.MustCompile(`(?i)^mesa[-_]?(\d+)$`)

func formateaID(id string) string {
	if m := reMesaNum.FindStringSubmatch(id); len(m) == 2 {
		return fmt.Sprintf("mesa-%02s", m[1])
	}
	return strings.ToLower(id)
}

func main() {
	mc, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017")) // cambiar localhost a 10.10.31.8 en MV2
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := mc.Connect(ctx); err != nil {
		log.Fatal(err)
	}
	db := mc.Database("tarea2-sd")
	mesasCol := db.Collection("mesas")

	regConn, err := grpc.Dial("localhost:50052", grpc.WithInsecure()) // cambiar localhost a IP de MV3 (10.10.31.9)
	if err != nil {
		log.Fatal(err)
	}
	regClient := registroPB.NewRegistroServiceClient(regConn)

	rabbitConn, err := amqp.Dial("amqp://guest:guest@localhost:5672/") // cambiar localhost a IP de MV1 (10.10.31.7)
	if err != nil {
		log.Println("RabbitMQ no disponible:", err)
	}
	var rabbitCh *amqp.Channel
	if rabbitConn != nil {
		rabbitCh, _ = rabbitConn.Channel()
		if rabbitCh != nil {
			_, _ = rabbitCh.QueueDeclare("reservas", false, false, false, false, nil)
		}
	}

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}
	grpcServer := grpc.NewServer()
	reservaPB.RegisterReservaServiceServer(grpcServer, &ReservaServer{dbMesas: mesasCol, registro: regClient, rabbitCh: rabbitCh})
	log.Println("✅ Servicio de reservas (MV2) escuchando en :50051")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
