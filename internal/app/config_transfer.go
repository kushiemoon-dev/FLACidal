package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	core "github.com/kushiemoon-dev/flacidal-core"
)

// ErrNotConfigExport marks a file that is not a FLACidal config export.
var ErrNotConfigExport = errors.New("not a FLACidal config export")

// secretFields lists the 8 credentials blanked on export. Single list so the
// export and the import groups below never diverge.
func secretFields(c *core.Config) []*string {
	return []*string{
		&c.TidalClientID, &c.TidalClientSecret,
		&c.SoulseekUsername, &c.SoulseekPassword,
		&c.QobuzAppID, &c.QobuzAppSecret, &c.QobuzAuthToken,
		&c.JellyfinAPIKey,
	}
}

// urlFields and urlLists name every config field that holds a URL, and so can
// carry a user:password part.
func urlFields(c *core.Config) []*string {
	return []*string{&c.JellyfinURL, &c.ProxyURL, &c.TidalCustomEndpoint, &c.QobuzCustomEndpoint}
}

func urlLists(c *core.Config) []*[]string {
	return []*[]string{
		&c.TidalHifiEndpoints, &c.QobuzEndpoints, &c.TidalPriorityEndpoints, &c.QobuzPriorityEndpoints,
		&c.AmazonPriorityEndpoints, &c.AmazonProxyEndpoints, &c.QobuzProxyEndpoints,
	}
}

// stripUserinfo drops the user:password part of a URL. A value that cannot be
// parsed, or contains an "@" that is not a userinfo separator, may still hold a
// password, so it is dropped too.
func stripUserinfo(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		if strings.Contains(raw, "@") {
			return ""
		}
		return raw
	}
	if u.User == nil {
		// An "@" with no userinfo parsed means a password the parser could not
		// see: "user:pass@host" and "socks5:u:p@h" parse as opaque, and a
		// password holding "/", "?" or "#" ends the authority early.
		if strings.Contains(raw, "@") {
			return ""
		}
		return raw
	}
	u.User = nil
	return u.String()
}

// destValue is a URL as compared between two configs: without credentials, but
// never empty for a non-empty value, so a value that had to be blanked (see
// stripUserinfo) is not mistaken for "no destination".
func destValue(v string) string {
	if s := stripUserinfo(v); s != "" || v == "" {
		return s
	}
	return v
}

func destValues(list []string) string {
	out := make([]string, len(list))
	for i, v := range list {
		out[i] = destValue(v)
	}
	return strings.Join(out, "\n")
}

// secretGroup is a set of credentials with the key of the destinations that
// receive them. Tidal client and Soulseek credentials are not sent to any
// configurable host, so their key is empty and they are always kept.
type secretGroup struct {
	fields []*string
	dest   string
}

func secretGroups(c *core.Config) []secretGroup {
	proxy := destValue(c.ProxyURL)
	return []secretGroup{
		{
			fields: []*string{&c.QobuzAppID, &c.QobuzAppSecret, &c.QobuzAuthToken},
			dest: strings.Join([]string{
				destValues(c.QobuzEndpoints), destValues(c.QobuzPriorityEndpoints), destValues(c.QobuzProxyEndpoints),
				destValue(c.QobuzCustomEndpoint), proxy,
			}, "\x00"),
		},
		{
			fields: []*string{&c.JellyfinAPIKey},
			dest:   destValue(c.JellyfinURL) + "\x00" + proxy,
		},
		{
			fields: []*string{&c.TidalClientID, &c.TidalClientSecret, &c.SoulseekUsername, &c.SoulseekPassword},
		},
	}
}

// BuildConfigExport serialises cfg as a schemaVersion 1 export with every
// secret blanked and the user and password removed from every URL. cfg is
// passed by value and its slices are copied, so the caller's config is untouched.
func BuildConfigExport(cfg core.Config) ([]byte, error) {
	for _, f := range secretFields(&cfg) {
		*f = ""
	}
	for _, f := range urlFields(&cfg) {
		*f = stripUserinfo(*f)
	}
	for _, l := range urlLists(&cfg) {
		if *l == nil {
			continue
		}
		stripped := make([]string, len(*l))
		for i, v := range *l {
			stripped[i] = stripUserinfo(v)
		}
		*l = stripped
	}
	data, err := json.MarshalIndent(configExport{SchemaVersion: 1, Config: cfg}, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("could not marshal config: %w", err)
	}
	return data, nil
}

// restoreURLCredentials puts back the user and password that the export
// removed, when the file's URL is the saved one without them. Credentials only
// go back to a host that is reached through the same proxy as before: with a
// new proxy in the file they would travel through it. The Jellyfin URL does not
// use the proxy, and the proxy URL is matched on its own.
func restoreURLCredentials(merged, current *core.Config) {
	proxySame := destValue(merged.ProxyURL) == destValue(current.ProxyURL)
	cur := urlFields(current)
	for i, f := range urlFields(merged) {
		own := f == &merged.ProxyURL || f == &merged.JellyfinURL
		if (own || proxySame) && *f != "" && *f == stripUserinfo(*cur[i]) {
			*f = *cur[i]
		}
	}
	if !proxySame {
		return
	}
	curLists := urlLists(current)
	for i, l := range urlLists(merged) {
		// Saved entries by stripped value, in order, so duplicates that differ
		// only by their credentials come back to the right slot.
		saved := map[string][]string{}
		for _, v := range *curLists[i] {
			k := stripUserinfo(v)
			saved[k] = append(saved[k], v)
		}
		for j, v := range *l {
			if q := saved[v]; len(q) > 0 {
				(*l)[j], saved[v] = q[0], q[1:]
			}
		}
	}
}

// MergeImportedConfig parses an export and returns the config to apply: the
// file's values, except that an empty secret keeps the one in current. A saved
// secret is only kept when the destinations that receive it (see secretGroups)
// are the same in the file; otherwise a shared file could point it at another server.
func MergeImportedConfig(data []byte, current core.Config) (core.Config, error) {
	var export configExport
	if err := json.Unmarshal(data, &export); err != nil {
		return core.Config{}, fmt.Errorf("could not parse import file: %w", ErrNotConfigExport)
	}
	if export.SchemaVersion != 1 {
		return core.Config{}, fmt.Errorf("not a FLACidal config export (schemaVersion %d): %w", export.SchemaVersion, ErrNotConfigExport)
	}
	merged := export.Config
	restoreURLCredentials(&merged, &current)

	curGroups := secretGroups(&current)
	for i, g := range secretGroups(&merged) {
		if g.dest != curGroups[i].dest {
			continue
		}
		for j, f := range g.fields {
			if *f == "" {
				*f = *curGroups[i].fields[j]
			}
		}
	}
	return merged, nil
}
