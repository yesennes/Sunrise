export TARGET=pi@10.0.0.129
export TARGET_DIR=/home/pi/Downloads/sunrise

.PHONY: build
build: tags
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

tags:
	(ctags -R . &)

clean:
	-rm Sunrise
	-rm PiSunrise
	-rm tags
