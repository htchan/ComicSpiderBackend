docker run --rm -d --name webhistory-sqlc-generator \
  -e POSTGRES_USER=web_history -e POSTGRES_PASSWORD=password -e POSTGRES_DB=db \
  -v ${PWD}/../migrations:/migrations -v ./:/sqlc/ postgres

# check if the container is ready
while ! docker exec -i webhistory-sqlc-generator pg_isready -U web_history; do sleep 1; done

# run migration and dump schema
docker exec webhistory-sqlc-generator bash -c 'for filename in /migrations/*.up.sql; do psql -U web_history -d db -f $filename; done' && \
docker exec webhistory-sqlc-generator bash -c "pg_dump -U web_history -d db -t websites -t user_websites -t website_settings --schema-only > /sqlc/schema.sql"

# kill container
docker kill webhistory-sqlc-generator