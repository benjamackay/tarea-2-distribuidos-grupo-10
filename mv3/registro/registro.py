# mv3/registro/registro.py
import logging
from concurrent import futures

import grpc
from pymongo import MongoClient

import registro_pb2
import registro_pb2_grpc


class RegistroServicer(registro_pb2_grpc.RegistroServiceServicer):
    def __init__(self, mongo_uri: str, db_name: str):
        self.client = MongoClient(mongo_uri)
        self.db = self.client[db_name]
        self.reservas_col = self.db["reservas"]
        self.mesas_col = self.db["mesas"]

    def RegistrarReserva(self, request, context):
        logging.info(
            "[Registro] reserva=%s name=%s mesa=%s status=%s",
            request.reservation_id,
            request.name,
            request.table_id,
            request.status,
        )

        try:
            # 1. guardar la reserva confirmada
            self.reservas_col.insert_one({
                "reservation_id": request.reservation_id,
                "name": request.name,
                "phone": request.phone,
                "party_size": request.party_size,
                "preferences": request.preferences,
                "table_id": request.table_id,
                "status": request.status,
            })

            # 2. actualizar mesa (solo si MV2 asignó una)
            if request.table_id:
                self.mesas_col.update_one(
                    {"table_id": request.table_id},
                    {"$set": {"status": "Reservada"}},
                )

            return registro_pb2.ConfirmacionRegistro(
                success=True,
                message="Reserva registrada correctamente",
            )
        except Exception as e:
            logging.exception("[Registro] error:")
            return registro_pb2.ConfirmacionRegistro(
                success=False,
                message=f"Error al registrar: {e}",
            )


def serve():
    # Mongo está en MV2
    mongo_uri = "mongodb://localhost:27017" # 10.10.31.8 o localhost
    db_name = "tarea2-sd"

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    registro_pb2_grpc.add_RegistroServiceServicer_to_server(
        RegistroServicer(mongo_uri, db_name), server
    )
    server.add_insecure_port("[::]:50052")
    logging.info("✅ registro.py escuchando en :50052")
    server.start()
    server.wait_for_termination()


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(asctime)s [%(levelname)s] %(message)s")
    serve()
