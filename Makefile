export GOOS=linux
export GOARCH=arm
export GOARM=5
.PHONY: build
build:
	go build

.PHONY: deploy
deploy: build
	scp psychic-pancake pi@lsenseney.com:/home/pi/Downloads/psychic-pancake

.PHONY: run
run: deploy
	ssh pi@lsenseney.com sudo /home/pi/Downloads/psychic-pancake

tags:
	ctags -R .
