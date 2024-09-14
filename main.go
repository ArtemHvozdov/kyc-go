package main

import (
	"encoding/json"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"net/http"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	circuits "github.com/iden3/go-circuits/v2"
	auth "github.com/iden3/go-iden3-auth/v2"

	"github.com/iden3/go-iden3-auth/v2/pubsignals"
	"github.com/iden3/go-iden3-auth/v2/state"
	// "github.com/iden3/iden3comm/v2/protocol"
)

const VerificationKeyPath = "verification_key.json"

type KeyLoader struct {
	Dir string
}

func (m KeyLoader) Load(id circuits.CircuitID) ([]byte, error) {
	return os.ReadFile(fmt.Sprintf("%s/%v/%s", m.Dir, id, VerificationKeyPath))
}

type TokenInfo struct {
    From    string `json:"from"`
    Message string `json:"message"`
}

func agentHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "success")
	Callback(w, r)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func main() {
	http.HandleFunc("/agent", agentHandler)
	http.HandleFunc("/status", statusHandler)

	fmt.Println("Server is running on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Server failed: %s\n", err)
	}
}

func Callback(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("sessionId")

	tokenBytes, err := io.ReadAll(r.Body)

	if err != nil {
		fmt.Println(err)
		return
	}

	ethURL := "AMOY_RPC_URL"

	// Add identity state contract address
	contractAddress := "0x1a4cC30f2aA0377b0c3bc9848766D90cb4404124"

	resolverPrefix := "polygon:amoy"

	// Locate the directory that contains circuit's verification keys
	keyDIR := "../keys"

	// fetch authRequest from sessionID
	authRequest := requestMap[sessionID]

	var verificationKeyLoader = &KeyLoader{Dir: keyDIR}
	resolver := state.ETHResolver{
		RPCUrl:          ethURL,
		ContractAddress: common.HexToAddress(contractAddress),
	}

	resolvers := map[string]pubsignals.StateResolver{
		resolverPrefix: resolver,
	}

	verifier, err := auth.NewVerifier(verificationKeyLoader, resolvers, auth.WithIPFSGateway("https://ipfs.io"))
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

	tokenStr := string(tokenBytes)
	
	parts := strings.Split(tokenStr, ".")
    if len(parts) != 3 {
        fmt.Println("Invalid token format")
        http.Error(w, "Invalid token format", http.StatusBadRequest)
        return
    }

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
    if err != nil {
        fmt.Println("Error decoding payload:", err)
        http.Error(w, "Invalid payload encoding", http.StatusBadRequest)
        return
    }

	var TargetedTokenInfo TokenInfo
    err = json.Unmarshal(payloadBytes, &TargetedTokenInfo)
    if err != nil {
        fmt.Println("Error unmarshalling payload:", err)
        return
    }

	fmt.Println("From:", TargetedTokenInfo.From)
    fmt.Println("Message:", TargetedTokenInfo.Message)


	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(messageBytes)
	fmt.Println("verification passed")
}