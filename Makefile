all: clean build

clean:
	rm -rf snowball
build:
	go build -v -o snowball
upx:
	upx -9 snowball
	rm -rf snowball.upx