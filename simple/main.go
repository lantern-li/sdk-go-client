package main

import (
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	"chainmaker.org/chainmaker/common/v2/json"
	sdkutils "chainmaker.org/chainmaker/sdk-go/v2/utils"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/common/v2/random/uuid"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	sdk "chainmaker.org/chainmaker/sdk-go/v2"
)

const (
	admin1CertPath = "./config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.crt"
	admin1KeyPath  = "./config/crypto-config/wx-org1.chainmaker.org/user/admin1/admin1.sign.key"
	admin2CertPath = "./config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.crt"
	admin2KeyPath  = "./config/crypto-config/wx-org2.chainmaker.org/user/admin1/admin1.sign.key"
	admin3CertPath = "./config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.crt"
	admin3KeyPath  = "./config/crypto-config/wx-org3.chainmaker.org/user/admin1/admin1.sign.key"
	admin5CertPath = "./config/crypto-config/wx-org5.chainmaker.org/user/admin1/admin1.sign.crt"
	admin5KeyPath  = "./config/crypto-config/wx-org5.chainmaker.org/user/admin1/admin1.sign.key"

	createContractTimeout    = 5
	claimContractName        = "claim003"
	claimVersion             = "2.0.0"
	claimByteCodePath        = "./config/rust-fact-2.0.0.wasm"
	sdkConfigOrg1Client1Path = "./config/sdk_config.yml"

	claimQueryMethod  = "find_by_file_hash"
	claimInvokeMethod = "save"

	org5Id         = "wx-org5.chainmaker.org"
	org5CaCertPath = "./config/crypto-config/wx-org5.chainmaker.org/ca/ca.crt"
)

func main() {
	//userContractClaim()
	//updateTrustRoot()
	//enableSyncCanonicalTxResult()
	oneTransactionPerSecond()
}

func userContractClaim() {
	fmt.Println("====================== create client ======================")

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	contract, err := client.GetContractInfo(claimContractName)
	if err != nil {
		if strings.Contains(err.Error(), "contract not exist") || contract == nil || contract.Name == "" {
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

	fmt.Println("====================== 调用合约 ======================")
	time.Sleep(time.Second * 2)
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
			Value: []byte(fmt.Sprintf("file_%s", curTime)),
		},
	}
	resp, err := client.InvokeContract(claimContractName, claimInvokeMethod, "", kvs, -1, true)
	panicErr(err)
	if resp.Code != common.TxStatusCode_SUCCESS {
		err = fmt.Errorf("invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
		panicErr(err)
	}
	txId := resp.TxId
	blockHeight := resp.TxBlockHeight
	fmt.Printf("blockHeight:%d, txId:%s, msg:%s, fileHash:%s\n\n",
		resp.TxBlockHeight, resp.TxId, resp.ContractResult.Message, fileHash)

	fmt.Println("====================== 执行合约查询接口 ======================")
	time.Sleep(time.Second * 2)
	kvs = []*common.KeyValuePair{
		{
			Key:   "file_hash",
			Value: []byte(fileHash),
		},
	}
	resp, err = client.QueryContract(claimContractName, claimQueryMethod, kvs, -1)
	panicErr(err)
	fmt.Printf("QUERY claim contract resp: %+v\n\n", resp)

	fmt.Println("====================== 执行交易查询接口 ======================")
	time.Sleep(time.Second * 2)
	tx, err := client.GetTxByTxId(txId)
	panicErr(err)
	fmt.Printf("%+v \n\n", tx)

	fmt.Println("====================== 执行区块查询接口 ======================")
	time.Sleep(time.Second * 2)
	block, err := client.GetBlockByHeight(blockHeight, false)
	panicErr(err)
	fmt.Printf("%+v \n\n", block)
}

func panicErr(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func updateTrustRoot() {
	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)
	config, err := client.GetChainConfig()
	panicErr(err)
	fmt.Printf("【before】\n%+v \n\n", config.GetTrustRoots())
	file, err := ioutil.ReadFile(org5CaCertPath)
	panicErr(err)

	// add trust root
	payload, err := client.CreateChainConfigTrustRootAddPayload(org5Id, []string{string(file)})
	panicErr(err)
	endorsers := make([]*common.EndorsementEntry, 0)
	endorser1, err1 := sdkutils.MakeEndorserWithPath(admin1KeyPath, admin1CertPath, payload)
	endorser2, err2 := sdkutils.MakeEndorserWithPath(admin2KeyPath, admin2CertPath, payload)
	endorser3, err3 := sdkutils.MakeEndorserWithPath(admin3KeyPath, admin3CertPath, payload)
	panicErr(err1)
	panicErr(err2)
	panicErr(err3)
	endorsers = append(endorsers, endorser1)
	endorsers = append(endorsers, endorser2)
	endorsers = append(endorsers, endorser3)
	_, err = client.SendChainConfigUpdateRequest(payload, endorsers, -1, true)
	panicErr(err)
	// query trust root
	config, err = client.GetChainConfig()
	fmt.Printf("【add】添加org5 ca\n%+v\n\n", config.GetTrustRoots())

	// invoke
	time.Sleep(time.Second * 2)
	resp, err := client.InvokeContractBySigner(claimContractName, claimQueryMethod, "", nil, -1,
		true, nil, getOrg5Signer())
	panicErr(err)
	fmt.Printf("\n【invoke】使用org5 ca签发证书执行成功，%s\n\n", resp.ContractResult.Message)

	// delete trust root
	payload, err = client.CreateChainConfigTrustRootDeletePayload(org5Id)
	panicErr(err)
	endorsers = make([]*common.EndorsementEntry, 0)
	endorser1, err1 = sdkutils.MakeEndorserWithPath(admin1KeyPath, admin1CertPath, payload)
	endorser2, err2 = sdkutils.MakeEndorserWithPath(admin2KeyPath, admin2CertPath, payload)
	endorser3, err3 = sdkutils.MakeEndorserWithPath(admin3KeyPath, admin3CertPath, payload)
	panicErr(err1)
	panicErr(err2)
	panicErr(err3)
	endorsers = append(endorsers, endorser1)
	endorsers = append(endorsers, endorser2)
	endorsers = append(endorsers, endorser3)
	_, err = client.SendChainConfigUpdateRequest(payload, endorsers, -1, true)
	panicErr(err)

	// query trust root
	config, err = client.GetChainConfig()
	fmt.Printf("【delete】删除org5 ca\n%+v\n\n", config.GetTrustRoots())

	// invoke
	time.Sleep(time.Second * 2)
	resp, err = client.InvokeContractBySigner(claimContractName, claimQueryMethod, "", nil, -1,
		true, nil, getOrg5Signer())
	panicErr(err)
	fmt.Printf("\n【invoke】 使用org5 ca签发证书执行失败，%s\n\n", resp.Message)
}

func getOrg5Signer() *sdk.CertModeSigner {
	certPem, err := ioutil.ReadFile(admin5CertPath)
	if err != nil {
		log.Fatalln(err)
	}
	cert, err := sdkutils.ParseCert(certPem)
	if err != nil {
		log.Fatalln(err)
	}
	privKeyPem, err := ioutil.ReadFile(admin5KeyPath)
	if err != nil {
		log.Fatalln(err)
	}
	privateKey, err := asym.PrivateKeyFromPEM(privKeyPem, nil)
	if err != nil {
		log.Fatalln(err)
	}

	signer := &sdk.CertModeSigner{
		PrivateKey: privateKey,
		Cert:       cert,
		OrgId:      org5Id,
	}
	return signer
}

func enableSyncCanonicalTxResult() {
	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
		sdk.WithEnableSyncCanonicalTxResult(true),
	)
	panicErr(err)

	fmt.Println("====================== 执行区块查询接口 ======================")
	tx, err := client.GetTxByTxId("177f8cc5d73cf3efcac808e9e1a8a8e07120d2aaeca74742a3d8c1a365c216bf")
	panicErr(err)
	txJson, _ := json.Marshal(tx)
	fmt.Printf("查询交易：%s", txJson)
}

func oneTransactionPerSecond() {

	client, err := sdk.NewChainClient(
		sdk.WithConfPath(sdkConfigOrg1Client1Path),
	)
	panicErr(err)

	fmt.Println("====================== 调用合约 ======================")
	for i := 0; i < 100; i++ {
		time.Sleep(time.Second * 1)
		t := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("%s 发送第%d个交易\n", t, i)
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
				Value: []byte(fmt.Sprintf("file_%s", curTime)),
			},
		}
		resp, err := client.InvokeContract(claimContractName, claimInvokeMethod, "", kvs, -1, false)
		panicErr(err)
		if resp.Code != common.TxStatusCode_SUCCESS {
			err = fmt.Errorf("invoke contract failed, [code:%d]/[msg:%s]\n", resp.Code, resp.Message)
			panicErr(err)
		}
		//txId := resp.TxId
		//blockHeight := resp.TxBlockHeight
		//t = time.Now().Format("2006-01-02 15:04:05")
		//fmt.Printf("%s blockHeight:%d, txId:%s, msg:%s, fileHash:%s\n\n",
		//	t, blockHeight, txId, resp.ContractResult.Message, fileHash)
	}
}
