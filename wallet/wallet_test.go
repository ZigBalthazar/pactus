package wallet

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"path"
	"testing"

	"github.com/pactus-project/pactus/genesis"
	"github.com/pactus-project/pactus/types/amount"
	"github.com/pactus-project/pactus/types/tx/payload"
	"github.com/pactus-project/pactus/util"
	"github.com/pactus-project/pactus/util/errors"
	"github.com/pactus-project/pactus/util/testsuite"
	pactus "github.com/pactus-project/pactus/www/grpc/gen/go"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

type testData struct {
	*testsuite.TestSuite

	server      *grpc.Server
	mockService *mockService
	wallet      *Wallet
	password    string
}

func setup(t *testing.T) *testData {
	t.Helper()

	ts := testsuite.NewTestSuite(t)

	password := ""
	walletPath := util.TempFilePath()
	mnemonic, _ := GenerateMnemonic(128)
	wallet, err := Create(walletPath, mnemonic, password, genesis.Mainnet)
	assert.NoError(t, err)
	assert.False(t, wallet.IsEncrypted())
	assert.Equal(t, wallet.Path(), walletPath)
	assert.Equal(t, wallet.Name(), path.Base(walletPath))

	// Mocking the gRPC server
	const bufSize = 1024 * 1024
	listener := bufconn.Listen(bufSize)

	server := grpc.NewServer()
	mockServer := &mockService{
		lastBlockHash:   ts.RandHash(),
		lastBlockHeight: ts.RandHeight(),
	}

	pactus.RegisterBlockchainServer(server, mockServer)
	pactus.RegisterTransactionServer(server, mockServer)

	go func() {
		if err := server.Serve(listener); err != nil {
			fmt.Printf("Server exited with error: %v", err)
		}
	}()

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}

	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("Failed to dial blockchain server: %v", err)
	}

	// Creating gRPC client
	client := &grpcClient{
		ctx:               context.Background(),
		blockchainClient:  pactus.NewBlockchainClient(conn),
		transactionClient: pactus.NewTransactionClient(conn),
	}

	wallet.client = client

	assert.False(t, wallet.IsEncrypted())
	assert.False(t, wallet.IsOffline())

	return &testData{
		TestSuite:   ts,
		server:      server,
		mockService: mockServer,
		wallet:      wallet,
		password:    password,
	}
}

func (ts *testData) Close() {
	ts.server.Stop()
	// TODO:  close client (wallet.Close() ??)
}

func TestOpenWallet(t *testing.T) {
	td := setup(t)
	defer td.Close()

	t.Run("Ok", func(t *testing.T) {
		assert.NoError(t, td.wallet.Save())
		_, err := Open(td.wallet.path, true)
		assert.NoError(t, err)
	})

	t.Run("Invalid wallet path", func(t *testing.T) {
		_, err := Open(util.TempFilePath(), true)
		assert.Error(t, err)
	})

	t.Run("Invalid crc", func(t *testing.T) {
		td.wallet.store.VaultCRC = 0
		bs, _ := json.Marshal(td.wallet.store)
		assert.NoError(t, util.WriteFile(td.wallet.path, bs))

		_, err := Open(td.wallet.path, true)
		assert.ErrorIs(t, err, CRCNotMatchError{
			Expected: td.wallet.store.calcVaultCRC(),
			Got:      0,
		})
	})

	t.Run("Invalid json", func(t *testing.T) {
		assert.NoError(t, util.WriteFile(td.wallet.path, []byte("invalid_json")))

		_, err := Open(td.wallet.path, true)
		assert.Error(t, err)
	})
}

func TestRecoverWallet(t *testing.T) {
	td := setup(t)
	defer td.Close()

	mnemonic, _ := td.wallet.Mnemonic(td.password)
	password := ""
	t.Run("Wallet exists", func(t *testing.T) {
		// Save the test wallet first then
		// try to recover a wallet at the same place
		assert.NoError(t, td.wallet.Save())

		_, err := Create(td.wallet.path, mnemonic, password, 0)
		assert.ErrorIs(t, err, ExitsError{
			Path: td.wallet.path,
		})
	})

	t.Run("Invalid mnemonic", func(t *testing.T) {
		_, err := Create(util.TempFilePath(),
			"invali mnemonic phrase seed", password, 0)
		assert.Error(t, err)
	})

	t.Run("Ok", func(t *testing.T) {
		walletPath := util.TempFilePath()
		recovered, err := Create(walletPath, mnemonic, password, genesis.Mainnet)
		assert.NoError(t, err)

		addr1, err := recovered.NewBLSAccountAddress("addr-1")
		assert.NoError(t, err)

		assert.NoFileExists(t, walletPath)
		assert.NoError(t, recovered.Save())

		assert.FileExists(t, walletPath)
		assert.True(t, recovered.Contains(addr1))
	})
}

func TestSaveWallet(t *testing.T) {
	td := setup(t)
	defer td.Close()

	t.Run("Invalid path", func(t *testing.T) {
		td.wallet.path = "/"
		assert.Error(t, td.wallet.Save())
	})
}

func TestInvalidAddress(t *testing.T) {
	td := setup(t)
	defer td.Close()

	addr := td.RandAccAddress().String()
	_, err := td.wallet.PrivateKey(td.password, addr)
	assert.Error(t, err)
}

func TestImportPrivateKey(t *testing.T) {
	td := setup(t)
	defer td.Close()

	_, prv := td.RandBLSKeyPair()
	assert.NoError(t, td.wallet.ImportPrivateKey(td.password, prv))

	pub := prv.PublicKeyNative()
	accAddr := pub.AccountAddress().String()
	valAddr := pub.AccountAddress().String()

	assert.True(t, td.wallet.Contains(accAddr))
	assert.True(t, td.wallet.Contains(valAddr))

	accAddrInfo := td.wallet.AddressInfo(accAddr)
	valAddrInfo := td.wallet.AddressInfo(accAddr)

	assert.Equal(t, pub.String(), accAddrInfo.PublicKey)
	assert.Equal(t, pub.String(), valAddrInfo.PublicKey)
}

func TestKeyInfo(t *testing.T) {
	td := setup(t)
	defer td.Close()

	mnemonic, _ := GenerateMnemonic(128)
	w1, err := Create(util.TempFilePath(), mnemonic, td.password,
		genesis.Mainnet)
	assert.NoError(t, err)
	addrStr1, _ := w1.NewBLSAccountAddress("")
	prv1, _ := w1.PrivateKey("", addrStr1)

	w2, err := Create(util.TempFilePath(), mnemonic, td.password,
		genesis.Testnet)
	assert.NoError(t, err)
	addrStr2, _ := w2.NewBLSAccountAddress("")
	prv2, _ := w2.PrivateKey("", addrStr2)

	assert.NotEqual(t, prv1.Bytes(), prv2.Bytes(),
		"Should generate different private key for the testnet")
}

func TestBalance(t *testing.T) {
	td := setup(t)
	defer td.Close()

	t.Run("existing account", func(t *testing.T) {
		amt, err := td.wallet.Balance(
			td.mockService.testAccountAddr.String())
		assert.NoError(t, err)
		assert.Equal(t, amount.Amount(1), amt)
	})

	t.Run("non-existing account", func(t *testing.T) {
		amt, err := td.wallet.Balance(
			td.RandAccAddress().String())
		assert.Error(t, err)
		assert.Zero(t, amt)
	})
}

func TestStake(t *testing.T) {
	td := setup(t)
	defer td.Close()

	t.Run("existing validator", func(t *testing.T) {
		amt, err := td.wallet.Stake(
			td.mockService.testValidatorAddr.String())
		assert.NoError(t, err)
		assert.Equal(t, amount.Amount(2), amt)
	})

	t.Run("non-existing validator", func(t *testing.T) {
		amt, err := td.wallet.Stake(
			td.RandValAddress().String())
		assert.Error(t, err)
		assert.Zero(t, amt)
	})
}

func TestSigningTx(t *testing.T) {
	td := setup(t)
	defer td.Close()

	sender, _ := td.wallet.NewBLSAccountAddress("testing addr")
	receiver := td.RandAccAddress()
	amt := td.RandAmount()
	fee := td.RandAmount()
	lockTime := td.RandHeight()

	opts := []TxOption{
		OptionFee(fee),
		OptionLockTime(lockTime),
		OptionMemo("test"),
	}

	trx, err := td.wallet.MakeTransferTx(sender, receiver.String(), amt, opts...)
	assert.NoError(t, err)
	err = td.wallet.SignTransaction(td.password, trx)
	assert.NoError(t, err)
	assert.NotNil(t, trx.Signature())
	assert.NoError(t, trx.BasicCheck())

	id, err := td.wallet.BroadcastTransaction(trx)
	assert.NoError(t, err)
	assert.Equal(t, trx.ID().String(), id)
	assert.Equal(t, fee, trx.Fee())
}

func TestMakeTransferTx(t *testing.T) {
	td := setup(t)
	defer td.Close()

	sender, _ := td.wallet.NewBLSAccountAddress("testing addr")
	receiver := td.RandAccAddress()
	amt := td.RandAmount()
	lockTime := td.RandHeight()

	t.Run("set parameters manually", func(t *testing.T) {
		fee := td.RandAmount()
		opts := []TxOption{
			OptionFee(fee),
			OptionLockTime(lockTime),
			OptionMemo("test"),
		}

		trx, err := td.wallet.MakeTransferTx(sender, receiver.String(), amt, opts...)
		assert.NoError(t, err)
		assert.Equal(t, fee, trx.Fee())
		assert.Equal(t, lockTime, trx.LockTime())
		assert.Equal(t, "test", trx.Memo())
	})

	t.Run("query parameters from the node", func(t *testing.T) {
		trx, err := td.wallet.MakeTransferTx(sender, receiver.String(), amt)
		assert.NoError(t, err)
		assert.Equal(t, trx.LockTime(), td.mockService.lastBlockHeight+1)
		assert.Equal(t, amt, trx.Payload().Value())
		fee, err := td.wallet.CalculateFee(amt, payload.TypeTransfer)
		assert.NoError(t, err)
		assert.Equal(t, fee, trx.Fee())
	})

	t.Run("invalid sender address", func(t *testing.T) {
		_, err := td.wallet.MakeTransferTx("invalid_addr_string", receiver.String(), amt)
		assert.Equal(t, errors.Code(err), errors.ErrInvalidAddress)
	})

	t.Run("invalid receiver address", func(t *testing.T) {
		_, err := td.wallet.MakeTransferTx(sender, "invalid_addr_string", amt)
		assert.Equal(t, errors.Code(err), errors.ErrInvalidAddress)
	})

	t.Run("unable to get the blockchain info", func(t *testing.T) {
		td.Close()

		_, err := td.wallet.MakeTransferTx(td.RandAccAddress().String(), receiver.String(), amt)
		assert.Equal(t, errors.Code(err), errors.ErrGeneric)
	})
}

func TestMakeBondTx(t *testing.T) {
	td := setup(t)
	defer td.Close()

	sender, _ := td.wallet.NewValidatorAddress("testing addr")
	receiver := td.RandValKey()
	amt := td.RandAmount()

	t.Run("set parameters manually", func(t *testing.T) {
		lockTime := td.RandHeight()
		fee := td.RandAmount()
		opts := []TxOption{
			OptionFee(fee),
			OptionLockTime(lockTime),
			OptionMemo("test"),
		}

		trx, err := td.wallet.MakeBondTx(sender, receiver.Address().String(),
			receiver.PublicKey().String(), amt, opts...)
		assert.NoError(t, err)
		assert.Equal(t, fee, trx.Fee())
		assert.Equal(t, lockTime, trx.LockTime())
		assert.Equal(t, "test", trx.Memo())
	})

	t.Run("query parameters from the node", func(t *testing.T) {
		trx, err := td.wallet.MakeBondTx(sender, receiver.Address().String(), receiver.PublicKey().String(), amt)
		assert.NoError(t, err)
		assert.Equal(t, trx.LockTime(), td.mockService.lastBlockHeight+1)
		assert.Equal(t, amt, trx.Payload().Value())
		fee, err := td.wallet.CalculateFee(amt, payload.TypeBond)
		assert.NoError(t, err)
		assert.Equal(t, fee, trx.Fee())
	})

	t.Run("validator address is not stored in wallet", func(t *testing.T) {
		t.Run("validator doesn't exist and public key not set", func(t *testing.T) {
			trx, err := td.wallet.MakeBondTx(sender, receiver.Address().String(), "", amt)
			assert.NoError(t, err)
			assert.Nil(t, trx.Payload().(*payload.BondPayload).PublicKey)
		})

		t.Run("validator doesn't exist and public key set", func(t *testing.T) {
			trx, err := td.wallet.MakeBondTx(sender, receiver.Address().String(), receiver.PublicKey().String(), amt)
			assert.NoError(t, err)
			assert.Equal(t, trx.Payload().(*payload.BondPayload).PublicKey.String(), receiver.PublicKey().String())
		})

		t.Run("validator exists and public key not set", func(t *testing.T) {
			trx, err := td.wallet.MakeBondTx(sender, receiver.Address().String(), "", amt)
			assert.NoError(t, err)
			assert.Nil(t, trx.Payload().(*payload.BondPayload).PublicKey)
		})

		t.Run("validator exists and public key set", func(t *testing.T) {
			trx, err := td.wallet.MakeBondTx(sender,
				td.mockService.testValidatorAddr.String(), receiver.PublicKey().String(), amt)
			assert.NoError(t, err)
			assert.Nil(t, trx.Payload().(*payload.BondPayload).PublicKey)
		})
	})

	t.Run("validator address stored in wallet", func(t *testing.T) {
		receiver, _ := td.wallet.NewValidatorAddress("validator-address")
		receiverInfo := td.wallet.AddressInfo(receiver)

		t.Run("validator doesn't exist and public key not set", func(t *testing.T) {
			trx, err := td.wallet.MakeBondTx(sender, receiver, "", amt)
			assert.NoError(t, err)
			assert.Equal(t, trx.Payload().(*payload.BondPayload).PublicKey.String(), receiverInfo.PublicKey)
		})

		t.Run("validator doesn't exist and public key set", func(t *testing.T) {
			receiverInfo := td.wallet.AddressInfo(receiver)
			trx, err := td.wallet.MakeBondTx(sender, receiver, receiverInfo.PublicKey, amt)
			assert.NoError(t, err)
			assert.Equal(t, trx.Payload().(*payload.BondPayload).PublicKey.String(), receiverInfo.PublicKey)
		})

		t.Run("validator exists and public key not set", func(t *testing.T) {
			trx, err := td.wallet.MakeBondTx(sender,
				td.mockService.testValidatorAddr.String(), "", amt)
			assert.NoError(t, err)
			assert.Nil(t, trx.Payload().(*payload.BondPayload).PublicKey)
		})

		t.Run("validator exists and public key set", func(t *testing.T) {
			receiverInfo := td.wallet.AddressInfo(receiver)
			trx, err := td.wallet.MakeBondTx(sender,
				td.mockService.testValidatorAddr.String(), receiverInfo.PublicKey, amt)
			assert.NoError(t, err)
			assert.Nil(t, trx.Payload().(*payload.BondPayload).PublicKey)
		})
	})

	t.Run("invalid sender address", func(t *testing.T) {
		_, err := td.wallet.MakeBondTx("invalid_addr_string", receiver.Address().String(), "", amt)
		assert.Equal(t, errors.Code(err), errors.ErrInvalidAddress)
	})

	t.Run("invalid receiver address", func(t *testing.T) {
		_, err := td.wallet.MakeBondTx(sender, "invalid_addr_string", "", amt)
		assert.Equal(t, errors.Code(err), errors.ErrInvalidAddress)
	})

	t.Run("invalid public key", func(t *testing.T) {
		_, err := td.wallet.MakeBondTx(sender, receiver.Address().String(), "invalid-pub-key", amt)
		assert.Equal(t, errors.Code(err), errors.ErrInvalidPublicKey)
	})

	t.Run("unable to get the blockchain info", func(t *testing.T) {
		td.Close()

		_, err := td.wallet.MakeBondTx(td.RandAccAddress().String(), receiver.Address().String(), "", amt)
		assert.Equal(t, errors.Code(err), errors.ErrGeneric)
	})
}

func TestMakeUnbondTx(t *testing.T) {
	td := setup(t)
	defer td.Close()

	sender, _ := td.wallet.NewValidatorAddress("testing addr")

	t.Run("set parameters manually", func(t *testing.T) {
		lockTime := td.RandHeight()
		opts := []TxOption{
			OptionLockTime(lockTime),
			OptionMemo("test"),
		}

		trx, err := td.wallet.MakeUnbondTx(sender, opts...)
		assert.NoError(t, err)
		assert.Zero(t, trx.Fee()) // Fee for unbond transaction is zero
		assert.Equal(t, lockTime, trx.LockTime())
		assert.Equal(t, "test", trx.Memo())
	})

	t.Run("query parameters from the node", func(t *testing.T) {
		trx, err := td.wallet.MakeUnbondTx(sender)
		assert.NoError(t, err)
		assert.Equal(t, trx.LockTime(), td.mockService.lastBlockHeight+1)
		assert.Zero(t, trx.Payload().Value())
		assert.Zero(t, trx.Fee())
	})

	t.Run("invalid sender address", func(t *testing.T) {
		_, err := td.wallet.MakeUnbondTx("invalid_addr_string")
		assert.Equal(t, errors.Code(err), errors.ErrInvalidAddress)
	})

	t.Run("unable to get the blockchain info", func(t *testing.T) {
		td.Close()

		_, err := td.wallet.MakeUnbondTx(td.RandAccAddress().String())
		assert.Equal(t, errors.Code(err), errors.ErrGeneric)
	})
}

func TestMakeWithdrawTx(t *testing.T) {
	td := setup(t)
	defer td.Close()

	sender, _ := td.wallet.NewBLSAccountAddress("testing addr")
	receiver, _ := td.wallet.NewBLSAccountAddress("testing addr")
	amt := td.RandAmount()

	t.Run("set parameters manually", func(t *testing.T) {
		lockTime := td.RandHeight()
		fee := td.RandAmount()
		opts := []TxOption{
			OptionFee(fee),
			OptionLockTime(lockTime),
			OptionMemo("test"),
		}

		trx, err := td.wallet.MakeWithdrawTx(sender, receiver, amt, opts...)
		assert.NoError(t, err)
		assert.Equal(t, fee, trx.Fee())
		assert.Equal(t, lockTime, trx.LockTime())
		assert.Equal(t, "test", trx.Memo())
	})

	t.Run("query parameters from the node", func(t *testing.T) {
		trx, err := td.wallet.MakeWithdrawTx(sender, receiver, amt)
		assert.NoError(t, err)
		assert.Equal(t, trx.LockTime(), td.mockService.lastBlockHeight+1)
		assert.Equal(t, amt, trx.Payload().Value())
		fee, err := td.wallet.CalculateFee(amt, payload.TypeWithdraw)
		assert.NoError(t, err)
		assert.Equal(t, fee, trx.Fee())
	})

	t.Run("invalid sender address", func(t *testing.T) {
		_, err := td.wallet.MakeWithdrawTx("invalid_addr_string", receiver, amt)
		assert.Equal(t, errors.Code(err), errors.ErrInvalidAddress)
	})

	t.Run("unable to get the blockchain info", func(t *testing.T) {
		td.Close()

		_, err := td.wallet.MakeWithdrawTx(td.RandAccAddress().String(), receiver, amt)
		assert.Equal(t, errors.Code(err), errors.ErrGeneric)
	})
}

func TestCheckMnemonic(t *testing.T) {
	mnemonic, _ := GenerateMnemonic(128)
	assert.NoError(t, CheckMnemonic(mnemonic))
}
