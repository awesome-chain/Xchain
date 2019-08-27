# Go TTC

Golang implementation of the TTC protocol.

[![GoDoc](https://img.shields.io/badge/godoc-reference-blue.svg)](https://godoc.org/github.com/awesome-chain/Xchain)
[![GoReport](https://goreportcard.com/badge/github.com/awesome-chain/Xchain)](https://goreportcard.com/report/github.com/awesome-chain/Xchain)
[![Travis](https://travis-ci.org/TTCECO/gttc.svg?branch=master)](https://travis-ci.org/TTCECO/gttc)
[![License](https://img.shields.io/badge/license-GPL%20v3-blue.svg)](LICENSE)
## About gttc

gttc is base on [go-ethereum](https://github.com/ethereum/go-ethereum), the main part be modified is in [consensus](consensus/) directory. We add a new consensus algorithm named [alien](consensus/alien/) in it.

Alien is a simple version of DPOS-PBFT consensus algorithm, which in [consensus/alien](consensus/alien/):

* **alien.go**          : Implement the consensus interface
* **custom_tx.go**      : Process the custom transaction such as vote,proposal,declare and so on...
* **snapshot.go**       : Keep the snapshot of vote and confirm status for each block
* **signer_queue.go**   : calculate the order of signer queue
* **api.go**            : API
* **cross_chain.go**    : Cross chain communication by custom transaction

Alien use header.extra to record the all infomation of current block and keep signature of miner. The snapshot keep vote & confirm information of whole chain, which will be update by each Seal or VerifySeal. By the end of each loop, the miner will calculate the next loop miners base on the snapshot. Code annotation will show the details about how it works.

## Mainnet Information
* **Current Mainnet and Testnet is deploy the code of branch release/v0.2.0**
* **Next version will be release on July 12, which contain the hard-fork at block height 2968888**
* **Please make sure your node upgrade to release/v0.2.0 before that block height.(before July 17,2019 UTC/GMT+8)**

More information about this upgrade will be found [UPGRADE TO GTTC V0.2.0](https://github.com/awesome-chain/Xchain/wiki/UPGRADE-TO-GTTC-V0.2.0)

## Minimum Requirements

Requirement|Notes
---|---
Go version | Go1.9 or higher

## Install

See the [HOWTO_INSTALL](https://github.com/awesome-chain/Xchain/wiki/Building-GTTC)

[Enode list for Mainnet & Slavenet](https://github.com/awesome-chain/Xchain/wiki/Public-Enode-address)

## Other Documents List

You can find all documents in our [Wiki](https://github.com/awesome-chain/Xchain/wiki/)

* [DPOS_CONSENSUS_ALGORITHM](https://github.com/awesome-chain/Xchain/wiki/DPOS_CONSENSUS_ALGORITHM): `description of DPOS algorithm`
* [PBFT_CONSENSUS_ALGORITHM](https://github.com/awesome-chain/Xchain/wiki/PBFT_CONSENSUS_ALGORITHM): `description of PBFT algorithm`
* [HOWTO_IMPLEMENT_DPOS_PBFT_IN_ALIEN](https://github.com/awesome-chain/Xchain/wiki/HOWTO_IMPLEMENT_DPOS_PBFT_IN_ALIEN): `details about how we implement dpos and pbft`
* [genesis.json](https://github.com/awesome-chain/Xchain/wiki/genesis.json)  : `genesis.json file for the testnet we deploy`
* [HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK](https://github.com/awesome-chain/Xchain/wiki/HOWTO_RUNNING_TEST_ON_PRIVATE_NETWORK) : `The instruction of deploy your own testnet`
* [HOWTO_VOTE_ON_GTTC](https://github.com/awesome-chain/Xchain/wiki//HOWTO_VOTE_ON_GTTC)  : `how to vote on alien testnet and view snapshot through API`
* [GENESIS_JSON_SAMPLE](https://github.com/awesome-chain/Xchain/wiki/GENESIS_JSON_SAMPLE) : `genesis.json sample`

* [build slave network instruction](https://github.com/awesome-chain/Xchain/wiki/build-slave-network-instruction)
* [how to check running status of slave network](https://github.com/awesome-chain/Xchain/wiki/how-to-check-running-status-of-slave-network)

## Contact

email: peng@ttc.eco
