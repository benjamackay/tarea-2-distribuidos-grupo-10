//inicializar mesas en MongoDB
db = db.getSiblingDB('restaurante');

db.mesas.drop();
db.reservas.drop();

//10 mesas con diferentes capacidades y tipos
db.mesas.insertMany([
    //4
    { table_id: "mesa-01", capacity: 4, status: "Disponible", tipo: "interior" },
    { table_id: "mesa-02", capacity: 4, status: "Disponible", tipo: "exterior" },
    { table_id: "mesa-03", capacity: 4, status: "Disponible", tipo: "fumadores" },
    { table_id: "mesa-04", capacity: 4, status: "Disponible", tipo: "no fumadores" },
    
    //8
    { table_id: "mesa-05", capacity: 8, status: "Disponible", tipo: "interior" },
    { table_id: "mesa-06", capacity: 8, status: "Disponible", tipo: "exterior" },
    { table_id: "mesa-07", capacity: 8, status: "Disponible", tipo: "fumadores" },
    { table_id: "mesa-08", capacity: 8, status: "Disponible", tipo: "no fumadores" },
    
    //12
    { table_id: "mesa-09", capacity: 12, status: "Disponible", tipo: "interior" },
    { table_id: "mesa-10", capacity: 12, status: "Disponible", tipo: "exterior" }
]);


print("Base de datos inicializada con 10 mesas");