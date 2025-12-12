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
	"net/http"
	"net/url"
	"os"
	"strings"

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

	apiBase := resolveAPIBase(*flagAPIBase)
	token, err := resolveToken()
	if err != nil {
		log.Fatalf("cannot resolve GitHub token: %v", err)
	}

	secretValue, err := resolveSecretValue(*flagValueEnv, *flagValue)
	if err != nil {
		log.Fatalf("cannot resolve secret value: %v", err)
	}

	if err := setRepoSecret(apiBase, token, *flagOwner, *flagRepo, *flagSecretName, secretValue); err != nil {
		log.Fatalf("failed to set secret: %v", err)
	}

	fmt.Printf("Secret %s updated for %s/%s\n", *flagSecretName, *flagOwner, *flagRepo)
}

func resolveAPIBase(flagValue string) string {
	candidates := []string{
		strings.TrimSpace(flagValue),
		strings.TrimSpace(os.Getenv("GITHUB_API_URL")),
	}

	for _, c := range candidates {
		if c != "" {
			return strings.TrimRight(c, "/")
		}
	}

	return "https://api.github.com"
}

func resolveToken() (string, error) {
	for _, name := range []string{"GITHUB_TOKEN", "GH_TOKEN"} {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v, nil
		}
	}
	return "", errors.New("no GitHub token found; set the GITHUB_TOKEN or GH_TOKEN environment variable with a personal access token (see https://github.com/settings/tokens)")
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

func setRepoSecret(apiBase, token, owner, repo, name, value string) error {
	pubKey, err := getRepoPublicKey(apiBase, token, owner, repo)
	if err != nil {
		return fmt.Errorf("get repo public key: %w", err)
	}

	encrypted, err := encryptWithPublicKey(pubKey.Key, value)
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}

	return putRepoSecret(apiBase, token, owner, repo, name, pubKey.ID, encrypted)
}

func getRepoPublicKey(apiBase, token, owner, repo string) (*repoPublicKey, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/actions/secrets/public-key", apiBase, owner, repo)

	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	addGitHubHeaders(req, token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API %s: %s", resp.Status, string(body))
	}

	var key repoPublicKey
	if err := json.NewDecoder(resp.Body).Decode(&key); err != nil {
		return nil, err
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

func putRepoSecret(apiBase, token, owner, repo, name, keyID, encryptedValue string) error {
	endpoint := fmt.Sprintf("%s/repos/%s/%s/actions/secrets/%s",
		apiBase, owner, repo, url.PathEscape(name))

	body, err := json.Marshal(secretPayload{
		EncryptedValue: encryptedValue,
		KeyID:          keyID,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	addGitHubHeaders(req, token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API %s: %s", resp.Status, string(b))
	}

	return nil
}

func addGitHubHeaders(req *http.Request, token string) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	if req.Header.Get("X-GitHub-Api-Version") == "" {
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
}
