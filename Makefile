#!/usr/bin/make -f

build:
	docker build --rm -f "Dockerfile" -t alignak:latest "."
	docker container create --name "temp" alignak:latest
	docker container cp temp:/data/alignak ./alignak
	docker container rm temp
