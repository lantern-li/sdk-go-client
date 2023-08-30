package main

import (
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
)

const (
	createContractTimeout    = 5
	claimContractName        = "claim003"
	claimVersion             = "2.0.0"
	claimByteCodePath        = "./config/rust-fact-2.0.0.wasm"
	sdkConfigOrg1Client1Path = "./config/sdk_config.yml"

	claimQueryMethod  = "find_by_file_hash"
	claimInvokeMethod = "save"
)

var (
	hashs = make(map[string]string, 0)
	l     sync.RWMutex
)

func main() {
	userContractClaim()
	//userUpdateCore()

}

func userUpdateCore() {
	fmt.Println("====================== create client ======================")

	c, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)
	payload, err := c.CreateChainConfigBlockUpdatePayload(true, 1000*60*30, 2000, 100, 400, 20)
	panicErr(err)
	endorsers := make([]*common.EndorsementEntry, 0)
	e1, err1 := sdkutils.MakeEndorserWithPath("../../test/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key", "../../test/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt", payload)
	e2, err2 := sdkutils.MakeEndorserWithPath("../../test/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key", "../../test/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt", payload)
	e3, err3 := sdkutils.MakeEndorserWithPath("../../test/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key", "../../test/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt", payload)
	panicErr(err1)
	panicErr(err2)
	panicErr(err3)
	endorsers = append(endorsers, e1)
	endorsers = append(endorsers, e2)
	endorsers = append(endorsers, e3)
	r, err := c.SendChainConfigUpdateRequest(payload, endorsers, 1000*10, true)
	panicErr(err)
	fmt.Println(r.Code, r.Message, r.ContractResult.Message)
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
			createPayload, err := client.CreateContractCreatePayload(claimContractName, claimVersion, claimByteCodePath, common.RuntimeType_WASMER, []*common.KeyValuePair{})
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
	for i := 0; i < 500000; i++ {
		//fmt.Println("====================== 调用合约 ======================")
		curTime := strconv.FormatInt(time.Now().Unix(), 10)
		fileHash := uuid.GetUUID()
		kvs := []*common.KeyValuePair{
			{
				Key:   "time",
				Value: []byte(curTime),
			},
			{
				Key:   "file_hash",
				Value: []byte(fileHash),
			},
			{
				Key:   "file_name",
				Value: []byte(fmt.Sprintf("file_file_file_file_file_file_file_file_file_file_file_file_file_file_file_file_file_file_file_file_%s", curTime)),
			},
		}
		_, err := client.InvokeContractBySigner(claimContractName, claimInvokeMethod, "", kvs, -1, false, nil, getSigner())
		panicErr(err)
		total++
		//txId := resp.TxId
		//tx, err := client.GetTxByTxId(txId)
		//panicErr(err)
		//fmt.Println(txId)
		//if tx != nil && tx.Transaction != nil && tx.Transaction.Sender != nil {
		//	cert := tx.Transaction.Sender.Signer.MemberInfo
		//	data, err := hash.Get(commonCrypto.HASH_TYPE_SHA256, cert)
		//	panicErr(err)
		//	hash := fmt.Sprintf("%x", data)
		//	l.Lock()
		//	hashs[hash] = hash
		//	l.Unlock()
		//
		//	fmt.Printf("txId %s singerHash256 %x singerHashCount:%d  \n", txId, data, len(hashs))
		//}

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
