// Copyright © 2021 Kaleido, Inc.
//
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ethereum

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/hyperledger/firefly/internal/config"
	"github.com/hyperledger/firefly/internal/config/wsconfig"
	"github.com/hyperledger/firefly/internal/i18n"
	"github.com/hyperledger/firefly/internal/log"
	"github.com/hyperledger/firefly/internal/restclient"
	"github.com/hyperledger/firefly/pkg/blockchain"
	"github.com/hyperledger/firefly/pkg/fftypes"
	"github.com/hyperledger/firefly/pkg/wsclient"
)

const (
	broadcastBatchEventSignature = "BatchPin(address,uint256,string,bytes32,bytes32,string,bytes32[])"
)

type Ethereum struct {
	ctx          context.Context
	topic        string
	instancePath string
	prefixShort  string
	prefixLong   string
	capabilities *blockchain.Capabilities
	callbacks    blockchain.Callbacks
	client       *resty.Client
	initInfo     struct {
		stream *eventStream
		subs   []*subscription
	}
	wsconn wsclient.WSClient
	closed chan struct{}
}

type eventStreamWebsocket struct {
	Topic string `json:"topic"`
}

type asyncTXSubmission struct {
	ID string `json:"id"`
}

type ethBatchPinInput struct {
	Namespace  string   `json:"namespace"`
	UUIDs      string   `json:"uuids"`
	BatchHash  string   `json:"batchHash"`
	PayloadRef string   `json:"payloadRef"`
	Contexts   []string `json:"contexts"`
}

type ethWSCommandPayload struct {
	Type  string `json:"type"`
	Topic string `json:"topic,omitempty"`
}

var requiredSubscriptions = []string{
	"BatchPin",
}

var addressVerify = regexp.MustCompile("^[0-9a-f]{40}$")

func (e *Ethereum) Name() string {
	return "ethereum"
}

func (e *Ethereum) Init(ctx context.Context, prefix config.Prefix, callbacks blockchain.Callbacks) (err error) {

	ethconnectConf := prefix.SubPrefix(EthconnectConfigKey)

	e.ctx = log.WithLogField(ctx, "proto", "ethereum")
	e.callbacks = callbacks

	if ethconnectConf.GetString(restclient.HTTPConfigURL) == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "url", "blockchain.ethconnect")
	}
	e.instancePath = ethconnectConf.GetString(EthconnectConfigInstancePath)
	if e.instancePath == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "instance", "blockchain.ethconnect")
	}
	e.topic = ethconnectConf.GetString(EthconnectConfigTopic)
	if e.topic == "" {
		return i18n.NewError(ctx, i18n.MsgMissingPluginConfig, "topic", "blockchain.ethconnect")
	}

	e.prefixShort = ethconnectConf.GetString(EthconnectPrefixShort)
	e.prefixLong = ethconnectConf.GetString(EthconnectPrefixLong)

	e.client = restclient.New(e.ctx, ethconnectConf)
	e.capabilities = &blockchain.Capabilities{
		GlobalSequencer: true,
	}

	wsConfig := wsconfig.GenerateConfigFromPrefix(ethconnectConf)

	if wsConfig.WSKeyPath == "" {
		wsConfig.WSKeyPath = "/ws"
	}

	e.wsconn, err = wsclient.New(ctx, wsConfig, e.afterConnect)
	if err != nil {
		return err
	}

	streams := streamManager{
		ctx:          e.ctx,
		client:       e.client,
		instancePath: e.instancePath,
	}
	batchSize := ethconnectConf.GetUint(EthconnectConfigBatchSize)
	batchTimeout := uint(ethconnectConf.GetDuration(EthconnectConfigBatchTimeout).Milliseconds())
	if e.initInfo.stream, err = streams.ensureEventStream(e.topic, batchSize, batchTimeout); err != nil {
		return err
	}
	log.L(e.ctx).Infof("Event stream: %s", e.initInfo.stream.ID)
	if e.initInfo.subs, err = streams.ensureSubscriptions(e.initInfo.stream.ID, requiredSubscriptions); err != nil {
		return err
	}

	e.closed = make(chan struct{})
	go e.eventLoop()

	return nil
}

func (e *Ethereum) Start() error {
	return e.wsconn.Connect()
}

func (e *Ethereum) Capabilities() *blockchain.Capabilities {
	return e.capabilities
}

func (e *Ethereum) afterConnect(ctx context.Context, w wsclient.WSClient) error {
	// Send a subscribe to our topic after each connect/reconnect
	b, _ := json.Marshal(&ethWSCommandPayload{
		Type:  "listen",
		Topic: e.topic,
	})
	err := w.Send(ctx, b)
	if err == nil {
		b, _ = json.Marshal(&ethWSCommandPayload{
			Type: "listenreplies",
		})
		err = w.Send(ctx, b)
	}
	return err
}

func ethHexFormatB32(b *fftypes.Bytes32) string {
	if b == nil {
		return "0x0000000000000000000000000000000000000000000000000000000000000000"
	}
	return "0x" + hex.EncodeToString(b[0:32])
}

func (e *Ethereum) handleBatchPinEvent(ctx context.Context, msgJSON fftypes.JSONObject) (err error) {
	sBlockNumber := msgJSON.GetString("blockNumber")
	sTransactionIndex := msgJSON.GetString("transactionIndex")
	sTransactionHash := msgJSON.GetString("transactionHash")
	dataJSON := msgJSON.GetObject("data")
	authorAddress := dataJSON.GetString("author")
	ns := dataJSON.GetString("namespace")
	sUUIDs := dataJSON.GetString("uuids")
	sBatchHash := dataJSON.GetString("batchHash")
	sPayloadRef := dataJSON.GetString("payloadRef")
	sContexts := dataJSON.GetStringArray("contexts")

	if sBlockNumber == "" ||
		sTransactionIndex == "" ||
		sTransactionHash == "" ||
		authorAddress == "" ||
		sUUIDs == "" ||
		sBatchHash == "" {
		log.L(ctx).Errorf("BatchPin event is not valid - missing data: %+v", msgJSON)
		return nil // move on
	}

	authorAddress, err = e.validateEthAddress(ctx, authorAddress)
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - bad from address (%s): %+v", err, msgJSON)
		return nil // move on
	}

	hexUUIDs, err := hex.DecodeString(strings.TrimPrefix(sUUIDs, "0x"))
	if err != nil || len(hexUUIDs) != 32 {
		log.L(ctx).Errorf("BatchPin event is not valid - bad uuids (%s): %+v", err, msgJSON)
		return nil // move on
	}
	var txnID fftypes.UUID
	copy(txnID[:], hexUUIDs[0:16])
	var batchID fftypes.UUID
	copy(batchID[:], hexUUIDs[16:32])

	var batchHash fftypes.Bytes32
	err = batchHash.UnmarshalText([]byte(sBatchHash))
	if err != nil {
		log.L(ctx).Errorf("BatchPin event is not valid - bad batchHash (%s): %+v", err, msgJSON)
		return nil // move on
	}

	contexts := make([]*fftypes.Bytes32, len(sContexts))
	for i, sHash := range sContexts {
		var hash fftypes.Bytes32
		err = hash.UnmarshalText([]byte(sHash))
		if err != nil {
			log.L(ctx).Errorf("BatchPin event is not valid - bad pin %d (%s): %+v", i, err, msgJSON)
			return nil // move on
		}
		contexts[i] = &hash
	}

	batch := &blockchain.BatchPin{
		Namespace:      ns,
		TransactionID:  &txnID,
		BatchID:        &batchID,
		BatchHash:      &batchHash,
		BatchPaylodRef: sPayloadRef,
		Contexts:       contexts,
	}

	// If there's an error dispatching the event, we must return the error and shutdown
	delete(msgJSON, "data")
	return e.callbacks.BatchPinComplete(batch, authorAddress, sTransactionHash, msgJSON)
}

func (e *Ethereum) handleReceipt(ctx context.Context, reply fftypes.JSONObject) error {
	l := log.L(ctx)

	headers := reply.GetObject("headers")
	requestID := headers.GetString("requestId")
	replyType := headers.GetString("type")
	txHash := reply.GetString("transactionHash")
	message := reply.GetString("errorMessage")
	if requestID == "" || replyType == "" {
		l.Errorf("Reply cannot be processed - missing fields: %+v", reply)
		return nil // Swallow this and move on
	}
	operationID, err := fftypes.ParseUUID(ctx, requestID)
	if err != nil {
		l.Errorf("Reply cannot be processed - bad ID: %+v", reply)
		return nil // Swallow this and move on
	}
	updateType := fftypes.OpStatusSucceeded
	if replyType != "TransactionSuccess" {
		updateType = fftypes.OpStatusFailed
	}
	l.Infof("Ethconnect '%s' reply: request=%s tx=%s message=%s", replyType, requestID, txHash, message)
	return e.callbacks.BlockchainOpUpdate(operationID, updateType, message, reply)
}

func (e *Ethereum) handleMessageBatch(ctx context.Context, messages []interface{}) error {
	l := log.L(ctx)

	for i, msgI := range messages {
		msgMap, ok := msgI.(map[string]interface{})
		if !ok {
			l.Errorf("Message cannot be parsed as JSON: %+v", msgI)
			return nil // Swallow this and move on
		}
		msgJSON := fftypes.JSONObject(msgMap)

		l1 := l.WithField("ethmsgidx", i)
		ctx1 := log.WithLogger(ctx, l1)
		signature := msgJSON.GetString("signature")
		l1.Infof("Received '%s' message", signature)
		l1.Tracef("Message: %+v", msgJSON)

		switch signature {
		case broadcastBatchEventSignature:
			if err := e.handleBatchPinEvent(ctx1, msgJSON); err != nil {
				return err
			}
		default:
			l.Infof("Ignoring event with unknown signature: %s", signature)
		}
	}

	return nil
}

func (e *Ethereum) eventLoop() {
	defer close(e.closed)
	l := log.L(e.ctx).WithField("role", "event-loop")
	ctx := log.WithLogger(e.ctx, l)
	ack, _ := json.Marshal(map[string]string{"type": "ack", "topic": e.topic})
	for {
		select {
		case <-ctx.Done():
			l.Debugf("Event loop exiting (context cancelled)")
			return
		case msgBytes, ok := <-e.wsconn.Receive():
			if !ok {
				l.Debugf("Event loop exiting (receive channel closed)")
				return
			}

			var msgParsed interface{}
			err := json.Unmarshal(msgBytes, &msgParsed)
			if err != nil {
				l.Errorf("Message cannot be parsed as JSON: %s\n%s", err, string(msgBytes))
				continue // Swallow this and move on
			}
			switch msgTyped := msgParsed.(type) {
			case []interface{}:
				err = e.handleMessageBatch(ctx, msgTyped)
				if err == nil {
					err = e.wsconn.Send(ctx, ack)
				}
			case map[string]interface{}:
				err = e.handleReceipt(ctx, fftypes.JSONObject(msgTyped))
			default:
				l.Errorf("Message unexpected: %+v", msgTyped)
				continue
			}

			// Send the ack - only fails if shutting down
			if err != nil {
				l.Errorf("Event loop exiting: %s", err)
				return
			}
		}
	}
}

func (e *Ethereum) ResolveSigningKey(ctx context.Context, signingKeyInput string) (signingKey string, err error) {
	return e.validateEthAddress(ctx, signingKeyInput)
}

func (e *Ethereum) validateEthAddress(ctx context.Context, identity string) (string, error) {
	identity = strings.TrimPrefix(strings.ToLower(identity), "0x")
	if !addressVerify.MatchString(identity) {
		return "", i18n.NewError(ctx, i18n.MsgInvalidEthAddress)
	}
	return "0x" + identity, nil
}

func (e *Ethereum) invokeContractMethod(ctx context.Context, method, signingKey string, requestID string, input interface{}, output interface{}) (*resty.Response, error) {
	return e.client.R().
		SetContext(ctx).
		SetQueryParam(e.prefixShort+"-from", signingKey).
		SetQueryParam(e.prefixShort+"-sync", "false").
		SetQueryParam(e.prefixShort+"-id", requestID).
		SetBody(input).
		SetResult(output).
		Post(e.instancePath + "/" + method)
}

func (e *Ethereum) SubmitBatchPin(ctx context.Context, operationID *fftypes.UUID, ledgerID *fftypes.UUID, signingKey string, batch *blockchain.BatchPin) error {
	tx := &asyncTXSubmission{}
	ethHashes := make([]string, len(batch.Contexts))
	for i, v := range batch.Contexts {
		ethHashes[i] = ethHexFormatB32(v)
	}
	var uuids fftypes.Bytes32
	copy(uuids[0:16], (*batch.TransactionID)[:])
	copy(uuids[16:32], (*batch.BatchID)[:])
	input := &ethBatchPinInput{
		Namespace:  batch.Namespace,
		UUIDs:      ethHexFormatB32(&uuids),
		BatchHash:  ethHexFormatB32(batch.BatchHash),
		PayloadRef: batch.BatchPaylodRef,
		Contexts:   ethHashes,
	}
	res, err := e.invokeContractMethod(ctx, "pinBatch", signingKey, operationID.String(), input, tx)
	if err != nil || !res.IsSuccess() {
		return restclient.WrapRestErr(ctx, res, err, i18n.MsgEthconnectRESTErr)
	}
	return nil
}
