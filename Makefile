export TARGET=pi@raspberrypi.local
export TARGET_DIR=/home/pi/Downloads/sunrise

.PHONY: build
build:
	go build

.PHONY: buildpi
buildpi: export GOOS=linux
buildpi: export GOARCH=arm
buildpi: export GOARM=6
buildpi: tags
	go build -o PiSunrise

.PHONY: runlocal
runlocal: build
	./Sunrise config.yaml

debuglocal: build
	dlv exec ./Sunrise config.yaml


.PHONY: deploy
deploy: buildpi
	rsync PiSunrise $(TARGET):$(TARGET_DIR)

.PHONY: run
run: deploy
	ssh $(TARGET) sudo $(TARGET_DIR)

tags: *.go
	(ctags -R . &)

clean:
	-rm Sunrise
	-rm PiSunrise
	-rm tags
