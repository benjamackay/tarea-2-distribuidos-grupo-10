package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"sort"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Estructura reservas
type SolicitudReserva struct {
	Name        string `json:"name"`
	Phone       string `json:"phone"`
	PartySize   int    `json:"party_size"`
	Preferences string `json:"preferences"`
}

// Estructura mesas
type Mesa struct {
	TableID  string `bson:"table_id"`
	Capacity int    `bson:"capacity"`
	Status   string `bson:"status"`
	Tipo     string `bson:"tipo"`
}

// Estructura resultados
type ResultadoReserva struct {
	Solicitud SolicitudReserva
	Mesa      *Mesa
	Exito     bool
	Mensaje   string
}

func main() {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect(ctx)

	db := client.Database("restaurante")
	coleccionMesas := db.Collection("mesas")

	reservasData, err := ioutil.ReadFile("../../reservas.json") //leer reservas
	if err != nil {
		log.Fatal(err)
	}

	var solicitudes []SolicitudReserva
	if err := json.Unmarshal(reservasData, &solicitudes); err != nil {
		log.Fatal(err)
	}

	//procesar solicitudes
	for _, solicitud := range solicitudes {
		resultado := asignarMesa(ctx, coleccionMesas, solicitud)
		if resultado.Exito {
			fmt.Printf("Reserva de %s para %d personas exitosa. Mesa asignada: %s (%d personas, %s)\n",
				resultado.Solicitud.Name, resultado.Solicitud.PartySize,
				resultado.Mesa.TableID, resultado.Mesa.Capacity, resultado.Mesa.Tipo)
		} else {
			fmt.Printf("Reserva de %s para %d personas fallida. Motivo: %s\n",
				resultado.Solicitud.Name, resultado.Solicitud.PartySize, resultado.Mensaje)
		}
	}
}

// asigna mesas segun reglas
func asignarMesa(ctx context.Context, coleccion *mongo.Collection, solicitud SolicitudReserva) ResultadoReserva {
	filter := bson.M{"status": "Disponible", "capacity": bson.M{"$gte": solicitud.PartySize}} //busca mesas
	cursor, err := coleccion.Find(ctx, filter)
	if err != nil {
		return ResultadoReserva{Solicitud: solicitud, Exito: false, Mensaje: "Error al consultar la base de datos"}
	}
	defer cursor.Close(ctx)

	var mesas []Mesa
	for cursor.Next(ctx) {
		var m Mesa
		if err := cursor.Decode(&m); err != nil {
			continue
		}
		//filtra preferencias
		if preferenciasExcluyentes(solicitud.Preferences, m.Tipo) {
			continue
		}
		mesas = append(mesas, m)
	}

	if len(mesas) == 0 {
		return ResultadoReserva{Solicitud: solicitud, Exito: false, Mensaje: "No hay mesas disponibles según preferencia y tamaño"}
	}

	//elegir mas pequeña
	sort.Slice(mesas, func(i, j int) bool {
		return mesas[i].Capacity < mesas[j].Capacity
	})

	mesaAsignada := mesas[0]

	//se cambia el estado de la mesa
	_, err = coleccion.UpdateOne(ctx, bson.M{"table_id": mesaAsignada.TableID}, bson.M{"$set": bson.M{"status": "Reservada"}})
	if err != nil {
		return ResultadoReserva{Solicitud: solicitud, Exito: false, Mensaje: "Error al actualizar mesa en la base de datos"}
	}

	return ResultadoReserva{Solicitud: solicitud, Mesa: &mesaAsignada, Exito: true}
}

// verificar preferencias excluyentes
func preferenciasExcluyentes(solicitud string, mesaTipo string) bool {
	if (solicitud == "fumadores" && mesaTipo == "no fumadores") || (solicitud == "no fumadores" && mesaTipo == "fumadores") {
		return true
	}
	if (solicitud == "interior" && mesaTipo == "exterior") || (solicitud == "exterior" && mesaTipo == "interior") {
		return true
	}
	return false
}
