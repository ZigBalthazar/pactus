package node

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zarbchain/zarb-go/account"
	"github.com/zarbchain/zarb-go/config"
	"github.com/zarbchain/zarb-go/crypto"
	"github.com/zarbchain/zarb-go/genesis"
	"github.com/zarbchain/zarb-go/util"
	"github.com/zarbchain/zarb-go/validator"
)

func TestRunningNode(t *testing.T) {
	_, pb, pv := crypto.RandomKeyPair()
	acc := account.NewAccount(crypto.MintbaseAddress)
	acc.SetBalance(21000000000000)
	val := validator.NewValidator(pb, 1)
	gen := genesis.MakeGenesis("test", time.Now(), []*account.Account{acc}, []*validator.Validator{val})
	conf := config.DefaultConfig()
	conf.State.Store.Path = util.TempDirName()
	conf.Network.NodeKey = util.TempFilename()
	privVal := validator.NewPrivValidator(pv)
	n, err := NewNode(gen, conf, privVal)

	assert.Equal(t, n.state.LastBlockHash(), crypto.UndefHash)

	require.NoError(t, err)
	err = n.Start()
	require.NoError(t, err)
	n.Stop()
}
