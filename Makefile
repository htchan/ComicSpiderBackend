.PHONY: backend frontend local_test backup test generate sqlc build

service ?= all

define setup_env
	$(eval ENV_FILE := ./data/env/.env.backend)
	@echo " - setup env $(ENV_FILE)"
	$(eval include ./data/env/.env.backend)
	$(eval export sed 's/=.*//' ./data/env/.env.backend)
endef

## help: show available command and description
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed  -e 's/^/ /'

## build service=<service>: build docker image of specified service (default all)
build:
	docker buildx bake backend -f docker-bake.hcl

clean-build:
	docker images --format "{{.Repository}}:{{.Tag}}" | \
		grep web-history | \
		xargs -L1 docker image rm

## backup the database content to ./bin/database
backup:
	docker compose --profile backup up --build --force-recreate

## api: deploy api container
api:
	docker compose --profile api up -d --force-recreate

migrate:
	${call setup_env}
	migrate -database 'postgres://${PSQL_USER}:${PSQL_PASSWORD}@${PSQL_HOST}:${PSQL_PORT}/${PSQL_DB}?sslmode=disable' -path ./database/migrations up

## worker: deploy worker container
worker:
	docker compose --profile worker up -d --force-recreate

start:
	docker compose pull api worker
	docker compose up -d api worker

test:
	go test ./... --cover --race --leak

bench:
	go test -bench=. -benchmem -benchtime=5s ./...

create_migrate:
	migrate create -ext sql -dir database/migrations $(NAME)

build-secrets:
	kubectl create secret generic web-history.api.secret --from-env-file ./data/env/.env.backend -o yaml --dry-run=client > deploy/api/secrets.yaml
	kubectl create secret generic web-history.worker.secret --from-env-file ./data/env/.env.worker -o yaml --dry-run=client > deploy/worker/secrets.yaml

PKG ?= "./..."

coverage:
	# go clean --testcache
	go test $(PKG) -coverprofile profile.txt ; go tool cover -html=profile.txt -o coverage.html
	rm profile.txt
	# google-chrome ./coverage.html &

sqlc:
	${call setup_env}
	PGPASSWORD=${PSQL_PASSWORD} pg_dump \
		-h ${PSQL_HOST} -p ${PSQL_PORT} -U ${PSQL_USER} -d ${PSQL_NAME} \
		-t websites -t user_websites -t website_settings --schema-only \
		> database/sqlc/schema.sql
	sqlc generate -f database/sqlc/sqlc.yaml
