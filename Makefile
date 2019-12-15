export TARGET=pi@raspberrypi.local
export TARGET_DIR=/home/pi/Downloads/sunrise

.PHONY: build
build: tags
	go build

.PHONY: build
buildpi: GOOS=linux GOARCH=arm GOARM=6
buildpi: tags
	go build

.PHONY: runlocal
runlocal: build
	./Sunrise config.yaml

debuglocal: build
	dlv exec ./Sunrise config.yaml


.PHONY: deploy
deploy: buildpi
	rsync psychic-pancake $(TARGET):$(TARGET_DIR)

.PHONY: run
run: deploy
	ssh $(TARGET) sudo $(TARGET_DIR)

tags:
	(ctags -R . &)
