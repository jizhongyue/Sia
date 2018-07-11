package api

import (
	"net/http"
	"encoding/json"
	"io/ioutil"
	"encoding/hex"

	"github.com/NebulousLabs/Sia/encoding"
	"github.com/NebulousLabs/Sia/types"

	"github.com/julienschmidt/httprouter"
)

type (
	// MinerGET contains the information that is returned after a GET request
	// to /miner.
	MinerGET struct {
		BlocksMined      int  `json:"blocksmined"`
		CPUHashrate      int  `json:"cpuhashrate"`
		CPUMining        bool `json:"cpumining"`
		StaleBlocksMined int  `json:"staleblocksmined"`
	}
	// MinerHeaderGET contains the information that is returned after a GET request
	// to /miner/header.
	MinerHeaderGet struct {
		Target           types.Target       `json:"target"`
		Height           types.BlockHeight  `json:"height"`
		Header           types.BlockHeader  `json:"header"`
	}
	// MinerBlockGET contains the information that is returned after a GET request
	// to /miner/block.
	MinerBlockGet struct {
		Target           types.Target       `json:"target"`
		Height           types.BlockHeight  `json:"height"`
		Block            types.Block        `json:"block"`
	}

	MinerBlockTemplateGet struct {
		Result types.BlockTemplate          `json:result`
	}

	MinerBlockSubmitPost struct {
		Header            types.BlockHeader   `json:header`
		Coinbase          []byte              `json:coinbase`
		MinerPayouts      [][]byte            `json:minerpayouts`
		Transactions      [][]byte            `json:transactions`
	}
)

// minerHandler handles the API call that queries the miner's status.
func (api *API) minerHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	blocksMined, staleMined := api.miner.BlocksMined()
	mg := MinerGET{
		BlocksMined:      blocksMined,
		CPUHashrate:      api.miner.CPUHashrate(),
		CPUMining:        api.miner.CPUMining(),
		StaleBlocksMined: staleMined,
	}
	WriteJSON(w, mg)
}

// minerStartHandler handles the API call that starts the miner.
func (api *API) minerStartHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	api.miner.StartCPUMining()
	WriteSuccess(w)
}

// minerStopHandler handles the API call to stop the miner.
func (api *API) minerStopHandler(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	api.miner.StopCPUMining()
	WriteSuccess(w)
}

// minerHeaderHandlerGET handles the API call that retrieves a block header
// for work.
func (api *API) minerHeaderHandlerGET(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	bhfw, target, height, err := api.miner.HeaderForWork()
	if err != nil {
		WriteError(w, Error{err.Error()}, http.StatusBadRequest)
		return
	}

	//w.Write(encoding.MarshalAll(target, bhfw))
	//
	// Modify by Joseph Yue, 2018-06-04.
	// For, /miner/header RPC interface return a json string.
	mth := MinerHeaderGet{
		Target: target,
		Height: height,
		Header: bhfw,
	}
	WriteJSON(w, mth)
}

// minerHeaderHandlerPOST handles the API call to submit a block header to the
// miner.
func (api *API) minerHeaderHandlerPOST(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var bh types.BlockHeader
	err := encoding.NewDecoder(req.Body).Decode(&bh)
	if err != nil {
		WriteError(w, Error{err.Error()}, http.StatusBadRequest)
		return
	}
	err = api.miner.SubmitHeader(bh)
	if err != nil {
		WriteError(w, Error{err.Error()}, http.StatusBadRequest)
		return
	}
	WriteSuccess(w)
}

// minerBlockHandlerGET handles the API call that retrieves a block
// for work.
func (api *API) minerBlockHandlerGET(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	//bfw, target, err := api.miner.HeaderForWork()
	bfw, target, height, err := api.miner.BlockForWork()
	if err != nil {
		WriteError(w, Error{err.Error()}, http.StatusBadRequest)
		return
	}
	//w.Write(encoding.MarshalAll(target, bfw)) /////// change to return json
	//
	// Modify by Joseph Yue, 2018-06-04.
	// For, /miner/block RPC interface return a json string.
	mtb := MinerBlockGet{
		Target: target,
		Height: height,
		Block:  bfw,
	}
	WriteJSON(w, mtb)
}

// minerBlockTemplateGET handles the API call that retrieves a block template
// for work.
// * /miner/getblocktemplate
func (api *API) minerBlockTemplateGET(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	gbt, err := api.miner.GetBlockTemplate()
	if err != nil {
		WriteError(w, Error{err.Error()}, http.StatusBadRequest)
		return
	}
	//w.Write(encoding.MarshalAll(target, bfw)) /////// change to return json
	//
	// Modify by Joseph Yue, 2018-06-04.
	// For, /miner/getblocktemplate RPC interface return a json string.
	result := MinerBlockTemplateGet {
		Result:  gbt,
	}
	WriteJSON(w, result)
}

// minerBlockHandlerPOST handles the API call to submit a block header to the
// miner.
func (api *API) minerBlockHandlerPOST(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {
	var mbsp MinerBlockSubmitPost
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		WriteError(w, Error{err.Error()}, http.StatusBadRequest)
		return
	}
	// err = json.Unmarshal(body, &b)
	err = json.Unmarshal(body, &mbsp)
	if err != nil {
		WriteError(w, Error{err.Error()}, http.StatusBadRequest)
		return
	}

	var b types.Block
	b.ParentID     = mbsp.Header.ParentID
	b.Nonce        = mbsp.Header.Nonce
	b.Timestamp    = mbsp.Header.Timestamp
	for _, payout := range mbsp.MinerPayouts {
		po := types.SiacoinOutput{}
		encoding.Unmarshal(payout, &po)
		b.MinerPayouts = append(b.MinerPayouts, po)
	}
	for _, transaction := range mbsp.Transactions {
		txn := types.Transaction{}
		encoding.Unmarshal(transaction, &txn)
		b.Transactions = append(b.Transactions, txn)
	}
	var coinbase types.Transaction
	coinb, err := hex.DecodeString(mbsp.Coinbase)
	if err != nil {
		encoding.Unmarshal(coinb, &coinbase)
		b.Transactions = append(b.Transactions, coinbase)
	}

	err = api.miner.SubmitBlock(b, mbsp.Header)
	if err != nil {
		WriteError(w, Error{err.Error()}, http.StatusBadRequest)
		return
	}
	WriteSuccess(w)
}