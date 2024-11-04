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
const ngrokURL = "https://a2e2-109-72-122-36.ngrok-free.app"
const issuerDID = "did:polygonid:polygon:amoy:2qQ68JkRcf3xrHPQPWZei3YeVzHPP58wYNxx2mEouR"
const agentURL = ngrokURL + "/agent"

var walletDID string
var userName string

type KeyLoader struct {
	Dir string
}

type sessionData struct {
	credential map[string]interface{}
	offer map[string]interface{}
	userName string
}

var firstUser sessionData

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

type ChangeHandlerToken struct {
	defaultPrivadoToken bool
	responseOfferPrivadoToken bool
}

var stateAgentHandlers = ChangeHandlerToken{}

//var requestMap = make(map[string]interface{})

func credentialIssuance(w http.ResponseWriter, r *http.Request) {
	log.Println("firstUser credential:", firstUser.credential)
	credentialJSON, _ := json.Marshal(firstUser.credential)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(credentialJSON)
}

func homehHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Server is runinng...")
}

func agentHandler(w http.ResponseWriter, r *http.Request) {
	if stateAgentHandlers.defaultPrivadoToken && !stateAgentHandlers.responseOfferPrivadoToken {
		log.Println("Function credentialIssuance start be calling...")
		log.Println("State agent handlers:", stateAgentHandlers)
		credentialIssuance(w,r)
		stateAgentHandlers.responseOfferPrivadoToken = true
		log.Println("Function credentialIssuance was be calling...")
		log.Println("State agent handlers:", stateAgentHandlers)
	}

	if !stateAgentHandlers.defaultPrivadoToken {
		GetInfoByToken(w,r)
		stateAgentHandlers.defaultPrivadoToken = true
		log.Println("State agent handlers:", stateAgentHandlers)
		log.Println("Function getInfoByToken was be calling...")
	}
}

func issueCredentialHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
        return
    }

    userName = r.FormValue("name")
    if userName == "" {
        http.Error(w, "Name is required", http.StatusBadRequest)
        return
    }

	firstUser.userName = userName

	log.Println("Name user:", userName)

	firstUser.credential, firstUser.offer = createCredentialAndOffer()

	http.Redirect(w, r, "/get-offer", http.StatusSeeOther)
}

func getOfferHandler(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "get-offer.html")
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
}

func getCredentialHandler(w http.ResponseWriter, r *http.Request) {
	credentialJSON, _ := json.Marshal(firstUser.credential)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(credentialJSON)
}

func main() {
	http.Handle("/agent/", http.StripPrefix("/agent/", http.FileServer(http.Dir("./"))))

	http.HandleFunc("/", homehHandler)
	http.HandleFunc("/agent", agentHandler)
	http.HandleFunc("/issue-credential", issueCredentialHandler)
	http.HandleFunc("/get-offer", getOfferHandler)
	http.HandleFunc("/offers", offersHandler)
	http.HandleFunc("/status", statusHandler)
	http.HandleFunc("/get-credential", getCredentialHandler)

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

	walletDID = from

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
						{
							"type": "KYCAgeCredential-test",
							"context": "https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/kyc-v4.jsonld",
						},
                    },
                    "type": "WebVerificationForm",
                    "url": ngrokURL + "/agent/webVerifiicationForm.html",
                    "expiration": time.Now().Add(24 * time.Hour).Format(time.RFC3339),
                    "description": "You can pass the verification on our KYC provider by following the next link",
                },
            },
        },
        "to": walletDID,
        "from": issuerDID,
    }

    msgBytes, _ := json.Marshal(proposal)
    return msgBytes
}

func createCredentialAndOffer() (map[string]interface{}, map[string]interface{}) {
	credential := map[string]interface{} {
		"id": "36f9e851-d713-4b50-8f8d-8a9382f138ca",
		"thid": "36f9e851-d713-4b50-8f8d-8a9382f138ca",
		"typ": "application/iden3comm-plain-json",
		"type": "https://iden3-communication.io/credentials/1.0/issuance-response",
		"to": walletDID,
		"from": issuerDID,
		"body": map[string]interface{} {
			"credential": map[string]interface{} {
				"id": "urn:uuid:53a608cb-b5b6-4cc9-96a8-c230ff955554",
				"@context": []string {
					"https://raw.githubusercontent.com/iden3/claim-schema-vocab/main/schemas/json-ld/kyc-v4.jsonld",
				},
				"type": []string {
					// "VerifiableCredential",
					// "KYCAgeCredential",
					"KYCAgeCredential-test",
				},
				"credentialSubject": map[string]interface{} {
					"id": "did:polygonid:polygon:mumbai:2qJUZDSCFtpR8QvHyBC4eFm6ab9sJo5rqPbcaeyGC4",
					"birthday": 19960424,
					"documentType": 99,
					"type": "KYCAgeCredential-test",
				},
				"issuer": issuerDID,
				"expirationDate": "2058-07-10T11:33:20.000Z",
				"issuanceDate":  time.Now().Format(time.RFC3339),
				"credentialSchema": map[string]interface{} {
					"id": "https://raw.githubusercontent.com/iden3/claim-schema-vocab/refs/heads/main/schemas/json/KYCAgeCredential-v4.json",
					"type": "JsonSchema2023",
				},
				"credentialStatus": map[string]interface{} {
					"id": "https://rhs-staging.polygonid.me",
					"revocationNonce": 1000,
					"type": "Iden3ReverseSparseMerkleTreeProof",
				},
				"proof": []map[string]interface{} {
					{
						"type": "BJJSignature2021",
						"issuerData": map[string]interface{} {
							"id": issuerDID,
							"state": map[string]interface{}{
								"rootOfRoots": "0000000000000000000000000000000000000000000000000000000000000000",
								"revocationTreeRoot": "0000000000000000000000000000000000000000000000000000000000000000",
								"claimsTreeRoot": "5e5dca21a62dfbbd8b984a997dd9d666b6710d5499906301b19fb6996bfc1a02",
								"value": "139042475daf67e0c340aef5540f8979b46a0eed13951c6ce60dad8875d1ab1e",
							},
							"authCoreClaim": "cca3371a6cb1b715004407e325bd993c0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000007cc77af06d6ea9f91434ce69e46821371cc341eb9e190262a16f46d15410ce0d1a090d38ded1e32a74e5fa0908764606adbcb917edff84b6ca60968d8a8d79090000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
							"mtp": map[string]interface{} {
								"existence": true,
								"siblings": []interface{}{},
							},
							"credentialStatus": map[string]interface{}{
								"id": "https://rhs-staging.polygonid.me",
								"revocationNonce": 0,
								"type": "Iden3ReverseSparseMerkleTreeProof",
							},
						},
						"coreClaim": "c9b2370371b7fa8b3dab2a5ba81b68382a00000000000000000000000000000002128459e997f3d9402b75af3fb9ad1e2db8f3288d3f5b6a13fa833621070d0069dee7de22463f48d75e219aed6cebcaadde18c1aacce28cd53db0317d1e9d2b0000000000000000000000000000000000000000000000000000000000000000e80300000000000080d481a60000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
						"signature": "4fb847963a0903ead57a45bcf511f84067b092aa5b21e95349ad3cf284f2678b8522765989e807ec7ab36ca541dab96a0604abc1a1ea27db1bf59a23f0686f04",
					},
				},
			},
		},
	}

	// log.Println("credential:", credential)

	credentialOffer :=  map[string]interface{} {
		"id": "36f9e851-d713-4b50-8f8d-8a9382f138ca",
		"thid": "36f9e851-d713-4b50-8f8d-8a9382f138ca",
		"typ": "application/iden3comm-plain-json",
		"type": "https://iden3-communication.io/credentials/1.0/offer",
		"body": map[string]interface{} {
		  "credentials": []map[string]interface{} {
				{
					"description": "KYCAgeCredential-test",
					"id": "c7b66a79-b930-49d1-9a97-66ab8fd792ac",
				},
		  },
		  "url": agentURL,
		},
		"to": walletDID,
		"from": issuerDID,
	}

	//log.Println("credentialOffer:", credentialOffer)

	return credential, credentialOffer
}

func offersHandler(w http.ResponseWriter, r *http.Request) {
	credentialOfferFormatJSON, _ := json.Marshal(firstUser.offer)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(credentialOfferFormatJSON)
}