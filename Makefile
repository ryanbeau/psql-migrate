.PHONY: build-docker
build-docker:
	docker build . -t psql-migrate -f build/package/migrate/DockerFile
