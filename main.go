package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	GetInfoByToken(w,r)
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func main() {
	http.Handle("/agent/", http.StripPrefix("/agent/", http.FileServer(http.Dir("./"))))

	// http.HandleFunc("/sign-in", GetAuthRequest)
	http.HandleFunc("/agent", agentHandler)
	http.HandleFunc("/callback", Callback)
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

	parts := strings.Split(tokenStr, ".")

	token := Token{
		header: parts[0], 
		payload: parts[1], 
		proof: parts[2],
	}

	payload, _ := base64.RawURLEncoding.DecodeString(token.payload)

	var payloadData map[string]interface{}

	err = json.Unmarshal(payload, &payloadData)
	if err != nil {
		fmt.Printf("Error parsing JSON: %v", err)
	}

	prettyPayload, _ := json.MarshalIndent(payloadData, "", "  ")

	formattedPayload := fmt.Sprintf("%s\n", prettyPayload)

	infoToken := InfoToken{}

	from, ok := payloadData["from"].(string)
	if !ok {
    	fmt.Println("Error: 'from' field is not a string")
    	return
	}

	infoToken.from = fmt.Sprintf("%v", from)
	infoToken.message = fmt.Sprintf("%v", formattedPayload)

	fmt.Println("From:", infoToken.from)
	fmt.Println("Message:", infoToken.message)

	credentialProposal := createCredentialProposal()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(credentialProposal)
}

func createCredentialProposal() []byte {
    proposal := map[string]interface{}{
        "id": "36f9e851-d713-4b50-8f8d-8a9382f138ca",
        "thid": "36f9e851-d713-4b50-8f8d-8a9382f138ca",
        "typ": "application/iden3comm-plain-json",
        "type": "https://iden3-communication.io/credentials/0.1/proposal",
        "body": map[string]interface{}{
            "proposals": []map[string]interface{}{
                {
                    "credentials": []map[string]interface{}{
                        {
                            "type": "LivenessProof",
                            "context": "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/kyc-v4.jsonld",
                        },
                        {
                            "type": "KYC",
                            "context": "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/kyc-v4.jsonld",
                        },
                    },
                    "type": "WebVerificationForm",
                    "url": "https://82be-185-208-113-238.ngrok-free.app/agent/index.html",
                    "expiration": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
                    "description": "You can pass the verification on our KYC provider by following the next link",
                },
            },
        },
        "to": "did:polygonid:polygon:mumbai:2qJUZDSCFtpR8QvHyBC4eFm6ab9sJo5rqPbcaeyGC4",
        "from": "did:iden3:polygon:mumbai:x3HstHLj2rTp6HHXk2WczYP7w3rpCsRbwCMeaQ2H2",
    }

    msgBytes, _ := json.Marshal(proposal)
    return msgBytes
}


func GetAuthRequest() []byte {

	// Audience is verifier id
	rURL := "https://82be-185-208-113-238.ngrok-free.app"
	sessionID := 1
	CallbackURL := "/callback"
	Audience := "did:polygonid:polygon:amoy:2qQ68JkRcf3xrHPQPWZei3YeVzHPP58wYNxx2mEouR"
	
	uri := fmt.Sprintf("%s%s?sessionId=%s", rURL, CallbackURL, strconv.Itoa(sessionID))
	
	// Generate request for basic authentication
	var request protocol.AuthorizationRequestMessage = auth.CreateAuthorizationRequest("test flow", Audience, uri)
	
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
		"context": "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/kyc-v4.jsonld",
		"type":    "KYCAgeCredential",
	}
	request.Body.Scope = append(request.Body.Scope, mtpProofRequest)
	
	// Store auth request in map associated with session ID
	requestMap[strconv.Itoa(sessionID)] = request
	
	// print request
	fmt.Println(request)
	
	msgBytes, _ := json.Marshal(request)
	
	return msgBytes
}


func Callback(w http.ResponseWriter, r *http.Request) {
    fmt.Println("callback")
    // Get session ID from request
    sessionID := r.URL.Query().Get("sessionId")

    // get JWZ token params from the post request
    tokenBytes, err := io.ReadAll(r.Body)
    if err != nil {
        log.Println(err)
        return
    }

    // Locate the directory that contains circuit's verification keys
    keyDIR := "./keys"

    // fetch authRequest from sessionID
    authRequest := requestMap[sessionID]

    // print authRequest
    log.Println(authRequest)

    // load the verifcation key
    var verificationKeyLoader = &KeyLoader{Dir: keyDIR}
	
	polygonAmoyResolver := state.ETHResolver{
		RPCUrl: "https://polygon-amoy.infura.io/v3/<API_KEY_INFURA>",
		ContractAddress: common.HexToAddress("0x1a4cC30f2aA0377b0c3bc9848766D90cb4404124"),
	}

	privadoMainResolver := state.ETHResolver{
		RPCUrl: "https://rpc-mainnet.privado.id",
		ContractAddress: common.HexToAddress("0x975556428F077dB5877Ea2474D783D6C69233742"),
	}

	resolvers := map[string]pubsignals.StateResolver{
		"plygon:amoy":polygonAmoyResolver,
		"privado:main":privadoMainResolver,
	}

    // EXECUTE VERIFICATION
    verifier, err := auth.NewVerifier(verificationKeyLoader, resolvers, auth.WithIPFSGateway("https://ipfs.io"))
    if err != nil {
        log.Println(err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }    
    authResponse, err := verifier.FullVerify(
        r.Context(),
        string(tokenBytes),
        authRequest.(protocol.AuthorizationRequestMessage),
        pubsignals.WithAcceptedStateTransitionDelay(time.Minute*5))
    if err != nil {
        log.Println(err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    //marshal auth resp
    messageBytes, err := json.Marshal(authResponse)
    if err != nil {
        log.Println(err.Error())
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Header().Set("Content-Type", "application/json")
    w.Write(messageBytes)
    log.Println("verification passed")
}
