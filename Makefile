IMAGE=sejvlond/kafkafeeder

kafkafeeder_*.deb:
	docker run \
		--rm \
		-it \
		-v `pwd`:/src \
		sejvlond/kafkafeeder_build

heka_*.deb:
	echo "Package heka and copy it here"
	exit 1

build: kafkafeeder_*.deb heka_*.deb
	docker build -t ${IMAGE} .

push: build
	docker push ${IMAGE}

clean:
	sudo rm -f *.deb
