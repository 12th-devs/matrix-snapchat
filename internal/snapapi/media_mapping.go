package snapapi

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

// Snapchat Web resolves chat media descriptors in-process with its bolt
// content resolver (9846a7958a5f0bee7197.js, ct = BoltResolver). The resolver
// downloads a public network mapping from bolt-gcdn.sc-cdn.net and feeds it
// to the WASM content resolver via updateNetworkMapping; the descriptor's
// readyLocationIds index into that mapping. Each mapping entry carries the
// bucket name, a storage tier, and URL arguments whose last element is the
// CDN path prefix. Proven live mappings:
//   - location 3  (bolt-prod-s3-us-east-2, tier 2)  -> cf-st.sc-cdn.net/c/{id}?uc=4   (message 4114)
//   - location 4  (bolt-prod-s3-us-east-1, tier 2)  -> cf-st.sc-cdn.net/d/{id}?uc=... (message 745)
//   - location 79 (bolt-prod-gcs-us-east-5, tier 1) -> bolt-gcdn.sc-cdn.net/bq/{id}?uc=4 (message 79,
//     verified through the authenticated Web session oracle and an
//     unauthenticated 200 fetch of the same URL)
//
// The mapping is public (no cookies or tokens; Web fetches it with plain
// fetch and Cache-Control max-age=86400).
const (
	boltNetworkMappingURL = "https://bolt-gcdn.sc-cdn.net/sc-bolt-nm-prod/network_mapping.bin"
	boltNetworkMappingTTL = 24 * time.Hour
	maxBoltMappingBytes   = 4 << 20

	// Storage tiers observed in mapping entries. Proven hosts: tier 2 (S3)
	// objects serve from cf-st.sc-cdn.net (live 4114/745/772) and tier 1
	// (GCS) objects serve from bolt-gcdn.sc-cdn.net (live 79).
	boltLocationTierGCS = 1
	boltLocationTierS3  = 2

	mediaCDNHostGCS = "bolt-gcdn.sc-cdn.net"
)

type boltLocation struct {
	Bucket     string
	Tier       uint64
	PathPrefix string
}

type boltNetworkMapping struct {
	locations map[uint64]boltLocation
}

// parseBoltNetworkMapping decodes the repeated bucket entries (top-level
// field 2) of the network mapping. Each entry: field 1 = location id,
// field 2 = bucket, field 4 = storage tier, field 5 = URL template, repeated
// field 6 = positional template arguments whose last element is the CDN path
// prefix (live entries: 3 -> "c", 4 -> "d", 79 -> "bq").
func parseBoltNetworkMapping(data []byte) (*boltNetworkMapping, error) {
	mapping := &boltNetworkMapping{locations: make(map[uint64]boltLocation)}
	err := parseProtoFields(data, func(field protowire.Number, kind protowire.Type, value []byte, varint uint64) error {
		if field != 2 || kind != protowire.BytesType {
			return nil
		}
		entry := struct {
			id      uint64
			hasID   bool
			bucket  string
			tier    uint64
			hasTier bool
			prefix  string
		}{}
		err := parseProtoFields(value, func(inner protowire.Number, innerKind protowire.Type, innerValue []byte, innerVarint uint64) error {
			switch {
			case inner == 1 && innerKind == protowire.VarintType:
				entry.id, entry.hasID = innerVarint, true
			case inner == 2 && innerKind == protowire.BytesType:
				entry.bucket = string(innerValue)
			case inner == 4 && innerKind == protowire.VarintType:
				entry.tier, entry.hasTier = innerVarint, true
			case inner == 6 && innerKind == protowire.BytesType:
				entry.prefix = string(innerValue)
			}
			return nil
		})
		if err != nil || !entry.hasID || !entry.hasTier {
			return nil
		}
		mapping.locations[entry.id] = boltLocation{
			Bucket:     entry.bucket,
			Tier:       entry.tier,
			PathPrefix: entry.prefix,
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid bolt network mapping: %w", err)
	}
	if len(mapping.locations) == 0 {
		return nil, fmt.Errorf("bolt network mapping has no location entries")
	}
	return mapping, nil
}

// mediaCDNURLFromLocation builds the deterministic CDN URL for one mapping
// entry, mirroring the proven Web resolver output shape
// https://{tier host}/{path prefix}/{contentId}?uc={useCase}.
func mediaCDNURLFromLocation(location boltLocation, contentID string, useCase uint64) (string, bool) {
	var host string
	switch location.Tier {
	case boltLocationTierGCS:
		host = mediaCDNHostGCS
	case boltLocationTierS3:
		host = mediaCDNHost
	default:
		return "", false
	}
	if !validMediaContentObjectID(contentID) || !validBoltPathPrefix(location.PathPrefix) {
		return "", false
	}
	u := url.URL{
		Scheme: "https",
		Host:   host,
		Path:   "/" + location.PathPrefix + "/" + contentID,
	}
	u.RawQuery = url.Values{mediaCDNUseCaseParam: {strconv.FormatUint(useCase, 10)}}.Encode()
	return u.String(), true
}

// validBoltPathPrefix constrains the mapping-derived path segment to short
// lowercase alphanumeric labels so the constructed URL cannot smuggle
// separators or query strings (observed prefixes: "a".."z", "bq", "mc").
func validBoltPathPrefix(prefix string) bool {
	if len(prefix) == 0 || len(prefix) > 8 {
		return false
	}
	for _, b := range []byte(prefix) {
		switch {
		case b >= 'a' && b <= 'z':
		case b >= '0' && b <= '9':
		default:
			return false
		}
	}
	return true
}

func (c *Client) fetchBoltNetworkMapping(ctx context.Context) (*boltNetworkMapping, error) {
	if c == nil || c.http == nil {
		return nil, fmt.Errorf("bolt network mapping fetch has no HTTP client")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, boltNetworkMappingURL, nil)
	if err != nil {
		return nil, err
	}
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("bolt network mapping fetch returned %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBoltMappingBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBoltMappingBytes {
		return nil, fmt.Errorf("bolt network mapping exceeds %d bytes", maxBoltMappingBytes)
	}
	return parseBoltNetworkMapping(data)
}

// boltLocationForDescriptor resolves the first usable readyLocationId of the
// descriptor through the cached network mapping. The mapping is fetched at
// most once per TTL; a missing location triggers one refresh, matching Web's
// periodic updateNetworkMapping behavior.
func (c *Client) boltLocationForDescriptor(ctx context.Context, descriptor MediaDescriptor) (boltLocation, bool, error) {
	if len(descriptor.ReadyLocationIDs) == 0 || descriptor.CDNObjectID == "" {
		return boltLocation{}, false, nil
	}
	c.mediaMappingMu.Lock()
	mapping := c.mediaMapping
	fresh := time.Since(c.mediaMappingAt) < boltNetworkMappingTTL
	c.mediaMappingMu.Unlock()

	var location boltLocation
	var found bool
	if mapping != nil {
		for _, id := range descriptor.ReadyLocationIDs {
			if candidate, ok := mapping.locations[id]; ok {
				location, found = candidate, true
				break
			}
		}
		if found || fresh {
			return location, found, nil
		}
	}

	fetched, err := c.fetchBoltNetworkMapping(ctx)
	if err != nil {
		return boltLocation{}, false, err
	}
	c.mediaMappingMu.Lock()
	c.mediaMapping = fetched
	c.mediaMappingAt = time.Now()
	c.mediaMappingMu.Unlock()
	for _, id := range descriptor.ReadyLocationIDs {
		if candidate, ok := fetched.locations[id]; ok {
			return candidate, true, nil
		}
	}
	return boltLocation{}, false, nil
}

// mediaMappedCDNURL returns the deterministic CDN URL derived from the bolt
// network mapping for descriptors the proven static /c/ and /d/ rules do not
// cover. ok=false means the mapping cannot serve this descriptor and the
// caller must fall back to the resolver RPC.
func (c *Client) mediaMappedCDNURL(ctx context.Context, descriptorBytes []byte) (string, bool, error) {
	descriptor := ParseMediaDescriptor(descriptorBytes)
	if !descriptor.Detected || descriptor.CDNObjectID == "" {
		return "", false, nil
	}
	location, ok, err := c.boltLocationForDescriptor(ctx, descriptor)
	if err != nil || !ok {
		return "", false, err
	}
	mappedURL, ok := mediaCDNURLFromLocation(location, descriptor.CDNObjectID, descriptor.UseCase)
	return mappedURL, ok, nil
}
