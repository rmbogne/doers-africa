.PHONY: migrate-up migrate-down migrate-version migrate-create

migrate-up:
	./scripts/migrate.sh up

migrate-down:
	./scripts/migrate.sh down

migrate-version:
	./scripts/migrate.sh version

migrate-create:
	@test -n "$(name)" || \
		(echo "Usage: make migrate-create name=description" && exit 1)
	migrate create \
		-ext sql \
		-dir db/migrations \
		-seq "$(name)"