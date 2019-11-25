export GOOS=linux
export GOARCH=arm
export GOARM=6
export TARGET=pi@lsenseney.com
export TARGET_DIR=/home/pi/Downloads/psychic-pancake

.PHONY: build
build:
	go build

.PHONY: deploy
deploy: build
	rsync psychic-pancake $(TARGET):$(TARGET_DIR)

.PHONY: run
run: deploy
	ssh $(TARGET) sudo $(TARGET_DIR)

tags:
	ctags -R .
