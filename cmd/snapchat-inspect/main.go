// snapchat-inspect is a read-only operator inspector for ONE Snapchat message.
// It reuses the connector's authenticated session and the production snapapi
// decode paths to dump the decoded envelope/contents structure, including
// ExternalMedia fields, media references, encryption metadata, and any text
// fields. It never writes state and never touches message rendering.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/colej/mautrix-snapchat/internal/config"
	sidecar "github.com/colej/mautrix-snapchat/internal/connector"
	"github.com/colej/mautrix-snapchat/internal/snapapi"
)

func main() {
	chatID := flag.String("chat", "", "conversation UUID (required)")
	messageID := flag.String("message", "", "Snapchat message ID (required)")
	fetchLimit := flag.Int("limit", 10, "message window size for locating the target")
	search := flag.String("search", "", "string to search for inside decoded contents")
	envFile := flag.String("env", "data/runtime/local-stack.env", "runtime env file with connector URL/secret")
	connectorURL := flag.String("connector-url", "", "connector base URL (overrides env file)")
	secret := flag.String("secret", "", "connector shared secret (overrides env file)")
	timeoutSeconds := flag.Int("timeout", 60, "overall timeout in seconds")
	flag.Parse()

	if strings.TrimSpace(*chatID) == "" || strings.TrimSpace(*messageID) == "" {
		fmt.Fprintln(os.Stderr, "both -chat and -message are required")
		os.Exit(2)
	}

	baseURL, sharedSecret := *connectorURL, *secret
	if baseURL == "" || sharedSecret == "" {
		fileURL, fileSecret, err := loadRuntimeEnv(*envFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "load runtime env: %v\n", err)
			os.Exit(1)
		}
		if baseURL == "" {
			baseURL = fileURL
		}
		if sharedSecret == "" {
			sharedSecret = fileSecret
		}
	}
	if baseURL == "" || sharedSecret == "" {
		fmt.Fprintln(os.Stderr, "connector base URL and shared secret are required (flags or env file)")
		os.Exit(2)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSeconds)*time.Second)
	defer cancel()

	sidecarClient := sidecar.New(config.ConnectorConfig{
		BaseURL:               baseURL,
		SharedSecret:          sharedSecret,
		RequestTimeoutSeconds: *timeoutSeconds,
	})
	auth, err := sidecarClient.APIAuth(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connector api-auth: %v\n", err)
		os.Exit(1)
	}
	if !auth.Authenticated {
		fmt.Fprintf(os.Stderr, "connector session not authenticated (state=%s)\n", auth.State)
		os.Exit(1)
	}
	if strings.TrimSpace(auth.CookieString) == "" || strings.TrimSpace(auth.SSOToken) == "" ||
		strings.TrimSpace(auth.SelfUserID) == "" || strings.TrimSpace(auth.MCSCOFIDsBin) == "" {
		fmt.Fprintf(os.Stderr, "connector api-auth incomplete: missing messenger session fields\n")
		os.Exit(1)
	}

	client, err := snapapi.New(snapapi.Config{
		CookieString:        auth.CookieString,
		SSOToken:            auth.SSOToken,
		SelfUserID:          auth.SelfUserID,
		UserAgent:           auth.BrowserUserAgent,
		SnapClientUserAgent: auth.SnapClientUserAgent,
		SecChUA:             auth.SecChUA,
		SecChUAPlatform:     auth.SecChUAPlatform,
		GRPCWebUserAgent:    auth.GRPCWebUserAgent,
		MCSCOFIDsBin:        auth.MCSCOFIDsBin,
		Timeout:             time.Duration(*timeoutSeconds) * time.Second,
		EELDecrypter:        &sidecarEELDecrypter{client: sidecarClient},
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "build snapapi client: %v\n", err)
		os.Exit(1)
	}
	if err := client.Authenticate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "authenticate: %v\n", err)
		os.Exit(1)
	}

	report, err := client.InspectMessage(ctx, *chatID, *messageID, *fetchLimit, *search)
	if err != nil {
		fmt.Fprintf(os.Stderr, "inspect message: %v\n", err)
		os.Exit(1)
	}
	printJSON(report)
}

func loadRuntimeEnv(path string) (baseURL, secret string, err error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return "", "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "export ")
		line = strings.TrimSpace(line)
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		value = strings.Trim(value, `"'`)
		switch key {
		case "SNAPCHAT_CONNECTOR_BASE_URL":
			baseURL = value
		case "SNAPCHAT_CONNECTOR_SHARED_SECRET", "SNAPCHAT_SHARED_SECRET":
			if secret == "" {
				secret = value
			}
		}
	}
	return baseURL, secret, nil
}

type sidecarEELDecrypter struct {
	client *sidecar.Client
	last   *sidecar.EELDecryptResponse
}

func (d *sidecarEELDecrypter) DecryptEEL(ctx context.Context, req snapapi.EELDecryptRequest) ([]byte, error) {
	if d.client == nil {
		return nil, fmt.Errorf("connector client is not initialized")
	}

	decrypted, response, err := d.client.DecryptEELDetailed(ctx, sidecar.EELDecryptRequest{
		ConversationID:        req.ConversationID,
		MessageID:             req.MessageID,
		ContentBase64:         base64Encode(req.Content),
		CEKBase64:             base64Encode(req.CEK),
		CEKIVBase64:           base64Encode(req.CEKIV),
		NonceBase64:           base64Encode(req.Nonce),
		SenderPublicKeyBase64: base64Encode(req.SenderPublicKey),
		SenderVersion:         req.SenderVersion,
	})
	d.last = response
	return decrypted, err
}

// LastEELDecryptDiagnostics exposes the connector's decrypt-attribution
// diagnostics for the most recent decrypt call so reports can prove the
// plaintext belongs to the requested message.
func (d *sidecarEELDecrypter) LastEELDecryptDiagnostics() *snapapi.EELDecryptDiagnostics {
	if d == nil || d.last == nil {
		return nil
	}
	return &snapapi.EELDecryptDiagnostics{
		Method:              d.last.Method,
		RequestedMessageID:  d.last.RequestedMessageID,
		MatchedWebMessageID: d.last.MatchedWebMessageID,
		ContentSource:       d.last.ContentSource,
		MessageCount:        d.last.MessageCount,
		ExactMatch:          d.last.ExactMatch,
	}
}
