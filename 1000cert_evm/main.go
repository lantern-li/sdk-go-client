package main

import (
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/evmutils/abi"
	"chainmaker.org/chainmaker/utils/v2"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
)

const (
	createContractTimeout    = 5
	claimContractName        = "proof_evm"
	claimVersion             = "2.0.0"
	claimByteCodePath        = "./config/Proof.bin"
	sdkConfigOrg1Client1Path = "./config/sdk_config.yml"
	abiPath                  = "./config/Proof.abi"

	claimInvokeMethod = "addRecord"
	claimQueryMethod  = "getRecord"
)

var (
	hashs = make(map[string]string, 0)
	l     sync.RWMutex
)

func main() {
	userContractClaim()

}

var total int64 = 0

func userContractClaim() {
	fmt.Println("====================== create client ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	contract, err := client.GetContractInfo(claimContractName)
	if err != nil {
		if strings.Contains(err.Error(), "contract not exist") {
			fmt.Printf("合约[%s]不存在\n", claimContractName)
			fmt.Println("====================== 创建合约 ======================")
			createPayload, err := client.CreateContractCreatePayload(claimContractName, claimVersion, claimByteCodePath, common.RuntimeType_EVM, []*common.KeyValuePair{})
			panicErr(err)
			resp, err := client.SendContractManageRequest(createPayload, nil, createContractTimeout, true)
			panicErr(err)
			fmt.Printf("blockHeight:%d, txId:%s, result:%s, msg:%s\n\n", resp.TxBlockHeight, resp.TxId, resp.ContractResult.Result, resp.ContractResult.Message)
		} else {
			panicErr(err)
		}
	} else {
		fmt.Printf("合约已存在 %+v \n\n", contract)
	}
	//
	go tps()
	for i := 0; i < 10; i++ {
		go send(client)
	}
	send(client)
}

func tps() {
	t := time.NewTicker(time.Second * 2)
	start := time.Now()
	for {
		select {
		case <-t.C:
			s := time.Since(start)
			fmt.Printf("tps %d total %d since %d \n", total*1000/s.Milliseconds(), total, s.Milliseconds()/1000)

			if s.Milliseconds()/1000 > 10 {
				start = time.Now()
				total = 0
				time.Sleep(time.Second * 3)
			}
		}
	}
}

func send(client *sdk.ChainClient) {
	for i := 0; i < 100000; i++ {
		//fmt.Println("====================== 调用合约 ======================")
		abiJson, err := ioutil.ReadFile(abiPath)
		if err != nil {
			log.Fatalln(err)
		}

		myAbi, err := abi.JSON(strings.NewReader(string(abiJson)))
		if err != nil {
			log.Fatalln(err)
		}

		signer := getSigner()
		uuid := utils.GetRandTxId()
		dataByte, err := myAbi.Pack(claimInvokeMethod, uuid, uuid)
		if err != nil {
			log.Fatalln(err)
		}
		dataString := hex.EncodeToString(dataByte)

		kvs := []*common.KeyValuePair{
			{
				Key:   "data",
				Value: []byte(dataString),
			},
		}

		// invoke
		_, err = client.InvokeContractBySigner(claimContractName, claimInvokeMethod, "", kvs, -1, false, nil, signer)
		panicErr(err)
		total++

		// invoke && query
		//resp, err := client.InvokeContractBySigner(claimContractName, claimInvokeMethod, "", kvs, -1, true, nil, signer)
		//panicErr(err)
		//total++
		//fmt.Printf("%+v \n\n", resp)
		//// query
		//dataByte, err = myAbi.Pack(claimQueryMethod, uuid)
		//if err != nil {
		//	log.Fatalln(err)
		//}
		//dataString = hex.EncodeToString(dataByte)
		//kvs = []*common.KeyValuePair{
		//	{
		//		Key:   "data",
		//		Value: []byte(dataString),
		//	},
		//}
		//resp, err = client.QueryContract(claimContractName, claimQueryMethod, kvs, -1)
		//fmt.Printf("%+v", resp)

	}
}

var (
	orgId    int64 = 0
	clientId int64 = 0
)

func getSigner() *sdk.CertModeSigner {
	oid := ((atomic.AddInt64(&orgId, 1)+1000)/1000)%4 + 1
	cid := (atomic.AddInt64(&clientId, 1) % 1000) + 1

	if oid > 4 {
		return nil
	}
	certPemPath := fmt.Sprintf("../../test/crypto-config/wx-org%d.chainmaker.org/user/client%d/client%d.sign.crt", oid, cid, cid)
	privateKeyPath := fmt.Sprintf("../../test/crypto-config/wx-org%d.chainmaker.org/user/client%d/client%d.sign.key", oid, cid, cid)
	//fmt.Println("orgId", oid, "privateKeyPath", privateKeyPath)

	certPem, err := ioutil.ReadFile(certPemPath)
	panicErr(err)

	cert, err := sdkutils.ParseCert(certPem)
	panicErr(err)

	privKeyPem, err := ioutil.ReadFile(privateKeyPath)
	panicErr(err)
	privateKey, err := asym.PrivateKeyFromPEM(privKeyPem, nil)
	panicErr(err)

	signer := &sdk.CertModeSigner{
		PrivateKey: privateKey,
		Cert:       cert,
		OrgId:      fmt.Sprintf("wx-org%d.chainmaker.org", oid),
	}
	return signer
}
func panicErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
