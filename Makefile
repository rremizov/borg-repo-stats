all: build/borg-repo-stats.Linux-armv5l build/borg-repo-stats.Linux-x86_64

build:
	mkdir build

build/borg-repo-stats.Linux-armv5l: build
	GOOS=linux GOARCH=arm GOARM=5 go build -o $@

build/borg-repo-stats.Linux-x86_64: build
	GOOS=linux GOARCH=amd64 go build -o $@

clean:
	rm -r build
