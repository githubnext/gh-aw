package main

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"golang.org/x/crypto/nacl/box"
)

type repoPublicKey struct {
	ID  string `json:"key_id"`
	Key string `json:"key"`
}

type secretPayload struct {
	EncryptedValue string `json:"encrypted_value"`
	KeyID          string `json:"key_id"`
}

func main() {
	var (
		flagOwner      = flag.String("owner", "", "GitHub repository owner or organization")
		flagRepo       = flag.String("repo", "", "GitHub repository name")
		flagSecretName = flag.String("secret", "", "Secret name to create or update")
		flagValue      = flag.String("value", "", "Secret value (if empty, read from stdin)")
		flagValueEnv   = flag.String("value-from-env", "", "Environment variable to read secret value from")
		flagAPIBase    = flag.String("api-url", "", "GitHub API base URL (default: https://api.github.com or $GITHUB_API_URL)")
	)

	flag.Parse()

	if *flagOwner == "" || *flagRepo == "" || *flagSecretName == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Create GitHub REST client using go-gh
	opts := api.ClientOptions{}
	if *flagAPIBase != "" {
		opts.Host = strings.TrimPrefix(strings.TrimPrefix(*flagAPIBase, "https://"), "http://")
	}
	client, err := api.NewRESTClient(opts)
	if err != nil {
		log.Fatalf("cannot create GitHub client: %v", err)
	}

	secretValue, err := resolveSecretValue(*flagValueEnv, *flagValue)
	if err != nil {
		log.Fatalf("cannot resolve secret value: %v", err)
	}

	if err := setRepoSecret(client, *flagOwner, *flagRepo, *flagSecretName, secretValue); err != nil {
		log.Fatalf("failed to set secret: %v", err)
	}

	fmt.Printf("Secret %s updated for %s/%s\n", *flagSecretName, *flagOwner, *flagRepo)
}

func resolveSecretValue(fromEnv, fromFlag string) (string, error) {
	if fromEnv != "" {
		v := os.Getenv(fromEnv)
		if v == "" {
			return "", fmt.Errorf("environment variable %s is not set or empty", fromEnv)
		}
		return v, nil
	}

	if fromFlag != "" {
		return fromFlag, nil
	}

	info, err := os.Stdin.Stat()
	if err != nil {
		return "", err
	}

	if info.Mode()&os.ModeCharDevice != 0 {
		fmt.Fprintln(os.Stderr, "Enter secret value, then press Ctrl+D:")
	}

	reader := bufio.NewReader(os.Stdin)
	var b strings.Builder

	for {
		line, err := reader.ReadString('\n')
		b.WriteString(line)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return "", err
		}
	}

	value := strings.TrimRight(b.String(), "\r\n")
	if value == "" {
		return "", errors.New("secret value is empty")
	}
	return value, nil
}

func setRepoSecret(client *api.RESTClient, owner, repo, name, value string) error {
	pubKey, err := getRepoPublicKey(client, owner, repo)
	if err != nil {
		return fmt.Errorf("get repo public key: %w", err)
	}

	encrypted, err := encryptWithPublicKey(pubKey.Key, value)
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}

	return putRepoSecret(client, owner, repo, name, pubKey.ID, encrypted)
}

func getRepoPublicKey(client *api.RESTClient, owner, repo string) (*repoPublicKey, error) {
	var key repoPublicKey
	path := fmt.Sprintf("repos/%s/%s/actions/secrets/public-key", owner, repo)
	if err := client.Get(path, &key); err != nil {
		return nil, fmt.Errorf("get public key: %w", err)
	}
	if key.ID == "" || key.Key == "" {
		return nil, errors.New("public key response missing key_id or key")
	}
	return &key, nil
}

func encryptWithPublicKey(publicKeyB64, plaintext string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return "", fmt.Errorf("decode public key: %w", err)
	}
	if len(raw) != 32 {
		return "", fmt.Errorf("unexpected public key length: %d", len(raw))
	}

	var pk [32]byte
	copy(pk[:], raw)

	ciphertext, err := box.SealAnonymous(nil, []byte(plaintext), &pk, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("nacl encryption failed: %w", err)
	}

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func putRepoSecret(client *api.RESTClient, owner, repo, name, keyID, encryptedValue string) error {
	path := fmt.Sprintf("repos/%s/%s/actions/secrets/%s", owner, repo, name)
	payload := secretPayload{
		EncryptedValue: encryptedValue,
		KeyID:          keyID,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return client.Put(path, strings.NewReader(string(body)), nil)
}
