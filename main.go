package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	circuits "github.com/iden3/go-circuits/v2"
	auth "github.com/iden3/go-iden3-auth/v2"

	// "github.com/iden3/iden3comm/protocol"

	"github.com/iden3/go-iden3-auth/v2/pubsignals"
	"github.com/iden3/go-iden3-auth/v2/state"
	"github.com/iden3/iden3comm/v2/protocol"
)

const VerificationKeyPath = "verification_key.json"

type KeyLoader struct {
	Dir string
}

func (m KeyLoader) Load(id circuits.CircuitID) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("%s/%v/%s", m.Dir, id, VerificationKeyPath))
}

type Token struct {
	header string
	payload string
	proof string
}

type InfoToken struct {
	from string
	message string
}

var requestMap = make(map[string]interface{})

func homehHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Server is runinng...")
}

func agentHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "success")
	Callback(w, r)
	GetInfoByToken(w,r)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func main() {
	http.HandleFunc("/agent", agentHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/", homehHandler)

	fmt.Println("Server is running on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed: %s\n", err)
	}
}

func GetInfoByToken(w http.ResponseWriter, r *http.Request) {
	tokenBytes, err := io.ReadAll(r.Body)

	if err != nil {
		fmt.Println(err)
		return
	}

	tokenStr := string(tokenBytes)

	fmt.Println(tokenStr)

	parts := strings.Split(tokenStr, ".")

	token := Token{
		header: parts[0], 
		payload: parts[1], 
		proof: parts[2],
	}

	payload, _ := base64.RawURLEncoding.DecodeString(token.payload)

	fmt.Println(payload)

	var payloadData map[string]interface{}

	err = json.Unmarshal(payload, &payloadData)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v", err)
	}

	infoToken := InfoToken{}

	from, ok := payloadData["from"]
	if !ok {
		fmt.Println("Error accessing body field")
		return
	}

	body, ok := payloadData["body"].(map[string]interface{})
	if !ok {
		fmt.Println("Error accessing body field")
		return
	}

	message, ok := body["message"]
	if message == nil {
		fmt.Println("Message field is nil")
	}
	if !ok {
		fmt.Println("Error accessing message field")
	}

	infoToken.from = fmt.Sprintf("%v", from)
	infoToken.message = fmt.Sprintf("%v", message)

	fmt.Println(infoToken)

}

func authRequest(w http.ResponseWriter, r *http.Request) {
	rURL := "NGROK URL"
	sessionID := 1
	CallbackURL := "/agent"
	Audience := "did:polygonid:polygon:amoy:2qQ68JkRcf3xrHPQPWZei3YeVzHPP58wYNxx2mEouR"


	uri := fmt.Sprintf("%s%s?sessionId=%s", rURL, CallbackURL, strconv.Itoa(sessionID))

	var request protocol.AuthorizationRequestMessage = auth.CreateAuthorizationRequest("test flow", Audience, uri)

	request.ID = "7f38a193-0918-4a48-9fac-36adfdb8b542"
	request.ThreadID = "7f38a193-0918-4a48-9fac-36adfdb8b542"

	// Add request for a specific proof
	var mtpProofRequest protocol.ZeroKnowledgeProofRequest
	mtpProofRequest.ID = 1
	mtpProofRequest.CircuitID = string(circuits.AtomicQuerySigV2CircuitID)
	mtpProofRequest.Query = map[string]interface{}{
		"allowedIssuers": []string{"*"},
		"credentialSubject": map[string]interface{}{
			"birthday": map[string]interface{}{
				"$lt": 20000101,
			},
		},
		"context": "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/kyc-v3.json-ld",
		"type":    "KYCAgeCredential",
	}
	request.Body.Scope = append(request.Body.Scope, mtpProofRequest)

	// Store auth request in map associated with session ID
	
	requestMap[strconv.Itoa(sessionID)] = request

	// print request
	fmt.Println(request)

	msgBytes, _ := json.Marshal(request)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(msgBytes)
	return
}


func Callback(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")

	tokenBytes, err := io.ReadAll(r.Body)
    if err != nil {
        fmt.Println(err)
        return
    }

	ipfsURL := "https://ipfs.io"

	resolverPrefix := "privado:test"

	keyDIR := "../keys"

	authRequest := requestMap[sessionID]

	var verificationKeyLoader = &KeyLoader{Dir: keyDIR}

	// privadoMainStateResolver := state.ETHResolver{
	// 	RPCUrl: "https://rpc-mainnet.privado.id",
	// 	ContractAddress: common.HexToAddress("0x975556428F077dB5877Ea2474D783D6C69233742"),
	// }

	privadoTestStateResolver := state.ETHResolver{
		RPCUrl: "https://rpc-testnet.privado.id/",
		ContractAddress: common.HexToAddress("0x975556428F077dB5877Ea2474D783D6C69233742"),
	}

	resolvers := map[string]pubsignals.StateResolver{
		resolverPrefix: privadoTestStateResolver,
	}
	


	verifier, err := auth.NewVerifier(verificationKeyLoader, resolvers, auth.WithIPFSGateway(ipfsURL))
    if err != nil {
        fmt.Println(err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

	authResponse, err := verifier.FullVerify(
        r.Context(),
        string(tokenBytes),
        authRequest.(protocol.AuthorizationRequestMessage),
        pubsignals.WithAcceptedStateTransitionDelay(time.Minute*5))
    if err != nil {
        fmt.Println(err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    //marshal auth resp
    messageBytes, err := json.Marshal(authResponse)
    if err != nil {
        fmt.Println(err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Header().Set("Content-Type", "application/json")
    w.Write(messageBytes)
    fmt.Println("verification passed")
}