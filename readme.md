# minitraefik
colima start
docker compose -f compose.gateway.yaml up -d --build
docker compose -f compose.apps.yaml up -d
docker compose -f compose.apps.yaml down
docker compose -f compose.gateway.yaml down