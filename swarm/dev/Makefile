.PHONY: build cluster test

default: build

build:
	go build -o bin/swarm    github.com/awesome-chain/Xchain/cmd/swarm
	go build -o bin/geth     github.com/awesome-chain/Xchain/cmd/geth
	go build -o bin/bootnode github.com/awesome-chain/Xchain/cmd/bootnode

cluster: build
	scripts/boot-cluster.sh

test:
	go test -v github.com/awesome-chain/Xchain/swarm/...
