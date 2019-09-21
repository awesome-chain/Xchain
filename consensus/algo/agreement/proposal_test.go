package agreement

import (
	"fmt"
	"github.com/awesome-chain/Xchain/crypto"
	"testing"
)

func TestProposal(t *testing.T) {
	k, _ := crypto.GenerateKey()
	b := crypto.FromECDSAPub(&k.PublicKey)
	fmt.Println(len(b))
	return
}
