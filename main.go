package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	//"strconv"
	"strings"
	"time"

	//"github.com/ethereum/go-ethereum/common"
	circuits "github.com/iden3/go-circuits/v2"
	//auth "github.com/iden3/go-iden3-auth/v2"

	// "github.com/iden3/iden3comm/protocol"

	//"github.com/iden3/go-iden3-auth/v2/pubsignals"
	//"github.com/iden3/go-iden3-auth/v2/state"
	//"github.com/iden3/iden3comm/v2/protocol"
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

func issueCredentialHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    name := r.FormValue("name")
    if name == "" {
        http.Error(w, "Name is required", http.StatusBadRequest)
        return
    }

	log.Println("Name user:", name)

    credential := map[string]interface{}{
        "id": "urn:uuid:53a608cb-b5b6-4cc9-96a8-c230ff955554",
        "@context": []string{
            "https://www.w3.org/2018/credentials/v1",
            "https://schema.iden3.io/core/jsonld/iden3proofs.jsonld",
        },
        "type": []string{"VerifiableCredential", "KYCAgeCredential"},
        "credentialSubject": map[string]interface{}{
            "id": "did:polygonid:polygon:mumbai:2qJUZDSCFtpR8QvHyBC4eFm6ab9sJo5rqPbcaeyGC4",
            "name": name,
            "birthday": 19960424,
        },
        "issuer": "did:iden3:polygon:mumbai:x3HstHLj2rTp6HHXk2WczYP7w3rpCsRbwCMeaQ2H2",
        "issuanceDate": time.Now().Format(time.RFC3339),
    }

	log.Println("credential:", credential)

    credentialJSON, err := json.Marshal(credential)
    if err != nil {
        http.Error(w, "Failed to create credential", http.StatusInternalServerError)
        return
    }

	log.Println("credentialJSON:", credentialJSON)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(credentialJSON)
}


func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func main() {
	http.Handle("/agent/", http.StripPrefix("/agent/", http.FileServer(http.Dir("./"))))

	http.HandleFunc("/", homehHandler)
	http.HandleFunc("/agent", agentHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/issue-credential", issueCredentialHandler)

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
                    "url": "https://25cd-185-208-113-238.ngrok-free.app/agent/index.html",
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
