package main

import (
	"net/http"
	"os"

	"github.com/ReidMason/plex-ani-sync/internal/plex"
	"github.com/ReidMason/plex-ani-sync/server"
	"github.com/ReidMason/plex-ani-sync/server/common"
	"github.com/ReidMason/plex-ani-sync/server/controllers/plexController"
)

const (
	appName          = "Plex Ani Sync"
	clientIdentifier = "plex-ani-sync-go-v2" // TODO: Generate a random client identifier and store it
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	plexAuth := plex.NewPlexAuth(clientIdentifier, appName, http.DefaultClient)

	controllers := []common.Controller{
		plexController.New(plexAuth),
	}
	server := server.New(controllers, http.DefaultServeMux, port)

	server.Start()
}

// pubKeyB64 := base64.RawURLEncoding.EncodeToString(pubKey)
// fmt.Printf("Public Key (base64url): %s\n\n", pubKeyB64)

// req, err := http.NewRequest("POST", "https://clients.plex.tv/api/v2/auth/jwk", nil)
// if err != nil {
// 	fmt.Println("Error creating request:", err)
// 	return
// }
// req.Header.Set("X-Plex-Client-Identifier", clientIdentifier)
// // req.Header.Set("X-Plex-Token", "your-existing-token")

// resp, err := http.DefaultClient.Do(req)
// if err != nil {
// 	fmt.Println("Error sending request:", err)
// 	return
// }
// defer resp.Body.Close()

// body, err := io.ReadAll(resp.Body)
// if err != nil {
// 	fmt.Println("Error reading response body:", err)
// 	return
// }

// fmt.Printf("Response: %s\n", string(body))

// type KeyPair struct {
// 	PrivateKey string `json:"private_key"`
// 	PublicKey  string `json:"public_key"`
// }

// func getKeyPair() (ed25519.PrivateKey, ed25519.PublicKey, error) {
// 	keyPairPath := filepath.Join("data", "keypair.json")

// 	// Create data directory if it doesn't exist
// 	if err := os.MkdirAll("data", 0755); err != nil {
// 		return nil, nil, fmt.Errorf("failed to create data directory: %w", err)
// 	}

// 	// Try to load existing keypair
// 	if data, err := os.ReadFile(keyPairPath); err == nil {
// 		var kp KeyPair
// 		if err := json.Unmarshal(data, &kp); err == nil {
// 			privKey, err := base64.StdEncoding.DecodeString(kp.PrivateKey)
// 			if err != nil {
// 				return nil, nil, fmt.Errorf("failed to decode private key: %w", err)
// 			}
// 			pubKey, err := base64.StdEncoding.DecodeString(kp.PublicKey)
// 			if err != nil {
// 				return nil, nil, fmt.Errorf("failed to decode public key: %w", err)
// 			}
// 			fmt.Println("Loaded existing keypair from", keyPairPath)
// 			return ed25519.PrivateKey(privKey), ed25519.PublicKey(pubKey), nil
// 		}
// 	}

// 	// Generate new keypair if file doesn't exist or is invalid
// 	pubKey, privKey, err := ed25519.GenerateKey(nil)
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("failed to generate keypair: %w", err)
// 	}

// 	// Save keypair to file
// 	kp := KeyPair{
// 		PrivateKey: base64.StdEncoding.EncodeToString(privKey),
// 		PublicKey:  base64.StdEncoding.EncodeToString(pubKey),
// 	}
// 	data, err := json.MarshalIndent(kp, "", "  ")
// 	if err != nil {
// 		return nil, nil, fmt.Errorf("failed to marshal keypair: %w", err)
// 	}

// 	if err := os.WriteFile(keyPairPath, data, 0600); err != nil {
// 		return nil, nil, fmt.Errorf("failed to save keypair: %w", err)
// 	}

// 	fmt.Println("Generated and saved new keypair to", keyPairPath)
// 	return privKey, pubKey, nil
// }
