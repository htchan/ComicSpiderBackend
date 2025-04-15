package database

// TODO: update following pg_dump command to:
// 1. create a new psql container
// 2. create a new database
// 3. run migration in database
// 4. run pg_dump command

//go:generate bash -c "./generate_schema.sh"
//go:generate sqlc generate -f sqlc.yaml
