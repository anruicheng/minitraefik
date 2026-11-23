# minitraefik
colima start
docker compose -f compose.gateway.yaml up -d --build
docker compose -f compose.apps.yaml up -d
curl 127.0.0.1:8080/users
curl 127.0.0.1:8080/orders
docker compose -f compose.apps.yaml down
docker compose -f compose.gateway.yaml down