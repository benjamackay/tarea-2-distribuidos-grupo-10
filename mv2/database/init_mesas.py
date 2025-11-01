import sys
from pymongo import MongoClient

MONGO_URI = "mongodb://10.10.31.8:27017/" # 10.10.31.8 o localhost
DB_NAME = "tarea2-sd"

def init_db():
    client = MongoClient(MONGO_URI)
    db = client[DB_NAME]

    db.mesas.drop()
    db.reservas.drop()

    mesas = [
        # 2 personas
        {"table_id": "mesa-01", "capacity": 2, "status": "Disponible", "tipo": "interior"},
        {"table_id": "mesa-02", "capacity": 2, "status": "Disponible", "tipo": "exterior"},
        # 4 personas
        {"table_id": "mesa-03", "capacity": 4, "status": "Disponible", "tipo": "interior"},
        {"table_id": "mesa-04", "capacity": 4, "status": "Disponible", "tipo": "exterior"},
        {"table_id": "mesa-05", "capacity": 4, "status": "Disponible", "tipo": "no fumadores"},
        # 8 personas
        {"table_id": "mesa-06", "capacity": 8, "status": "Disponible", "tipo": "interior"},
        {"table_id": "mesa-07", "capacity": 8, "status": "Disponible", "tipo": "exterior"},
        {"table_id": "mesa-08", "capacity": 8, "status": "Disponible", "tipo": "fumadores"},
        # 12 personas
        {"table_id": "mesa-09", "capacity": 12, "status": "Disponible", "tipo": "interior"},
        {"table_id": "mesa-10", "capacity": 12, "status": "Disponible", "tipo": "fumadores"},
    ]

    db.mesas.insert_many(mesas)
    print("✅ Base de datos inicializada con 10 mesas.")

def drop_db():
    client = MongoClient(MONGO_URI)
    client.drop_database(DB_NAME)
    print("🗑️ Base de datos eliminada exitosamente.")

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Uso: python init_mesas.py [init|drop]")
        sys.exit(1)

    command = sys.argv[1].lower()
    if command == "init":
        init_db()
    elif command == "drop":
        drop_db()
    else:
        print("Comando no válido. Usa 'init' o 'drop'.")
