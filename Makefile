build:
	go build -o proxy.o proxy.go
	go build -o dataNode.o dataNode.go

clean:
	rm -rf proxy.o dataNode.o
