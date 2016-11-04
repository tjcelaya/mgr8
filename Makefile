BUILDSTAMP = $(shell date -u +"%Y%m%dT%H%M%SZ")

clean: db-clean build-clean

build: build-mac build-linux

build-mac:
	go build -o mgr8-mac-${BUILDSTAMP} -ldflags "-X main.buildstamp=${BUILDSTAMP}" main.go

build-linux:
	env GOOS=linux GOARCH=amd64 go build -o mgr8-linux-${BUILDSTAMP} -ldflags "-X main.buildstamp=${BUILDSTAMP}" main.go

build-clean:
	rm mgr8-*

build-db:
	docker-compose up -d

db-clean: TMP = $(shell docker ps -qa --filter=name=mgr8)
db-clean:
	docker stop ${TMP} && docker rm ${TMP}

db-client:
	mysql -uroot -psecret -h127.0.0.1 mgr8_db
