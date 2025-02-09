package goclient

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/artela-network/galxe-integration/config"
	store "github.com/artela-network/galxe-integration/goclient/contract"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

var globalaAddress string = ""

func TestDeployContract(t *testing.T) {
	c, err := NewClient("http://47.251.61.27:8545")
	require.Equal(t, nil, err)
	defer c.Close()

	privKey, pubKey, err := ReadKey("../privateKey.txt")
	require.Equal(t, nil, err)

	cfg := &config.TxConfig{}
	cfg.FillDefaults()

	fromAddress := crypto.PubkeyToAddress(*pubKey)
	opts := c.DefaultTxOpts(privKey, fromAddress, cfg)
	nonce, err := c.PendingNonceAt(context.Background(), fromAddress)
	opts.Nonce = big.NewInt(int64(nonce))
	// input := "1.0"

	address, tx, instance, err := store.DeployStorage(opts, c)
	require.Equal(t, nil, err)
	require.Equal(t, 32, len(address))
	globalaAddress = address.Hex()
	require.Equal(t, true, tx != nil)
	require.Equal(t, true, instance != nil)
}

func TestLoadContract(t *testing.T) {
	TestDeployContract(t)

	c, err := NewClient("http://47.251.61.27:8545")
	require.Equal(t, nil, err)
	defer c.Close()

	address := common.HexToAddress(globalaAddress)
	instance, err := store.NewStorage(address, c)
	require.Equal(t, nil, err)
	require.Equal(t, true, instance != nil)
}

func TestSend(t *testing.T) {
	c, err := NewClient("http://47.251.61.27:8545")
	require.Equal(t, nil, err)
	defer c.Close()

	privKey, pubKey, err := ReadKey("../privateKey.txt")
	require.Equal(t, nil, err)

	// deploy contract
	fromAddress := crypto.PubkeyToAddress(*pubKey)
	cfg := &config.TxConfig{}
	cfg.FillDefaults()
	opts := c.DefaultTxOpts(privKey, fromAddress, cfg)
	nonce, err := c.PendingNonceAt(context.Background(), fromAddress)
	require.Equal(t, nil, err)
	opts.Nonce = big.NewInt(int64(nonce)) // we maintance the nonce ourself

	_, _, instance, err := store.DeployStorage(opts, c)
	require.Equal(t, nil, err)
	time.Sleep(2 * time.Second)

	// send a tx
	opts = c.DefaultTxOpts(privKey, fromAddress, cfg)
	opts.Nonce = big.NewInt(int64(nonce + 1))

	storeTx, err := instance.Store(opts, "wang", store.StoragePerson{
		Id:      222,
		Balance: 5000,
	})
	require.Equal(t, nil, err)
	require.Equal(t, true, storeTx != nil)
	require.Equal(t, true, storeTx.Hash().Hex() != common.Hash{}.Hex())

	time.Sleep(2 * time.Second)

	{

		// try to query this tx
		tx, isPending, err := c.QueryTxByHash(context.Background(), storeTx.Hash())
		require.Equal(t, nil, err)
		require.Equal(t, false, isPending)

		json, err := json.Marshal(tx)
		require.Equal(t, nil, err)
		fmt.Println(string(json)) //
	}

	{
		// try to get the receipt
		receipt, err := c.TransactionReceipt(context.Background(), storeTx.Hash())
		require.Equal(t, nil, err)
		json, err := json.Marshal(receipt)
		require.Equal(t, nil, err)
		fmt.Println(string(json)) // {"root":"0x","status":"0x1","cumulativeGasUsed":"0x249f0","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","logs":[],"transactionHash":"0xb52e3c6750173bc19390fb79e25aa96194294394291a69a3283f9890fc76f280","contractAddress":"0x0000000000000000000000000000000000000000","gasUsed":"0x493e0","effectiveGasPrice":null,"blockHash":"0x431264894b4228738a5771f38184006820fe1770044ded69e142b0c4c094fca0","blockNumber":"0x1f9413","transactionIndex":"0x0"}
		require.Equal(t, ethtypes.ReceiptStatusSuccessful, receipt.Status)
	}
}

func TestCreate(t *testing.T) {
	pubpath := "./_address.txt"
	privpath := "./_keys.txt"

	pubfile, err := os.Open(pubpath)
	if err != nil && os.IsNotExist(err) {
		pubfile, err = os.Create(pubpath)
		require.Equal(t, nil, err)
	}

	privfile, err := os.Open(privpath)
	if err != nil && os.IsNotExist(err) {
		privfile, err = os.Create(privpath)
		require.Equal(t, nil, err)
	}

	var pubstr, privstr strings.Builder
	for i := 1; i <= 2000000; i++ {
		privateKey, err := crypto.GenerateKey()
		if err != nil {
			log.Fatal(err)
		}

		publicKey := privateKey.Public()
		publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
		require.Equal(t, true, ok)

		pub := crypto.PubkeyToAddress(*publicKeyECDSA).Hex()
		privateKeyBytes := crypto.FromECDSA(privateKey)
		priv := hexutil.Encode(privateKeyBytes)
		fmt.Println("privKey:", priv, "pubKey:", pub, "index:", i)
		pubstr.WriteString(strconv.Itoa(i))
		pubstr.WriteString(": ")
		pubstr.WriteString(pub)
		pubstr.WriteString("\n")

		privstr.WriteString(strconv.Itoa(i))
		privstr.WriteString(": ")
		privstr.WriteString(priv)
		privstr.WriteString("\n")

		if i%1000 == 0 {
			_, err = privfile.WriteString(privstr.String())
			require.Equal(t, nil, err)

			_, err = pubfile.WriteString(pubstr.String())
			require.Equal(t, nil, err)

			privstr.Reset()
			pubstr.Reset()
		}
	}
	_, err = privfile.WriteString(privstr.String())
	require.Equal(t, nil, err)

	_, err = pubfile.WriteString(pubstr.String())
	require.Equal(t, nil, err)
}
