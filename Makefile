all: build/borg-repo-stats.linux-amd64 build/borg-repo-stats.linux-arm5

build:
	mkdir build

build/borg-repo-stats.linux-arm5: build
	GOOS=linux GOARCH=arm GOARM=5 go build -o $@

build/borg-repo-stats.linux-amd64: build
	GOOS=linux GOARCH=amd64 go build -o $@

clean:
	rm -r build
