package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sensu-community/sensu-plugin-sdk/sensu"
	corev2 "github.com/sensu/sensu-go/api/core/v2"
	"github.com/sensu/sensu-go/types"
)

// Config represents the check plugin config.
type Config struct {
	sensu.PluginConfig
	CheckLabels        string
	EntityLabels       string
	Namespaces         string
	APIHost            string
	APIPort            int
	APIUser            string
	APIPass            string
	APIKey             string
	Secure             bool
	TrustedCAFile      string
	InsecureSkipVerify bool
	Protocol           string
	WarnPercent        int
	CritPercent        int
	WarnCount          int
	CritCount          int
}

// Auth represents the authentication info
type Auth struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// Counters represents the analyzed components and statuses count
type Counters struct {
	Entities int
	Checks   int
	Ok       int
	Warning  int
	Critical int
	Unknown  int
	Total    int
}

var (
	tlsConfig tls.Config

	plugin = Config{
		PluginConfig: sensu.PluginConfig{
			Name:     "sensu-aggregate-check",
			Short:    "The Sensu Go Event Aggregates Check plugin",
			Keyspace: "sensu.io/plugins/sensu-aggregate-check/config",
		},
	}

	options = []*sensu.PluginConfigOption{
		&sensu.PluginConfigOption{
			Path:      "check-labels",
			Env:       "",
			Argument:  "check-labels",
			Shorthand: "l",
			Default:   "",
			Usage:     "Sensu Go Event Check Labels to filter by (e.g. 'aggregate=foo')",
			Value:     &plugin.CheckLabels,
		},
		&sensu.PluginConfigOption{
			Path:      "entity-labels",
			Env:       "",
			Argument:  "entity-labels",
			Shorthand: "e",
			Default:   "",
			Usage:     "Sensu Go Event Entity Labels to filter by (e.g. 'aggregate=foo,app=bar')",
			Value:     &plugin.EntityLabels,
		},
		&sensu.PluginConfigOption{
			Path:      "namespaces",
			Env:       "",
			Argument:  "namespaces",
			Shorthand: "n",
			Default:   "default",
			Usage:     "Comma-delimited list of Sensu Go Namespaces to query for Events (e.g. 'us-east-1,us-west-2')",
			Value:     &plugin.Namespaces,
		},
		&sensu.PluginConfigOption{
			Path:      "api-host",
			Env:       "",
			Argument:  "api-host",
			Shorthand: "H",
			Default:   "127.0.0.1",
			Usage:     "Sensu Go Backend API Host (e.g. 'sensu-backend.example.com')",
			Value:     &plugin.APIHost,
		},
		&sensu.PluginConfigOption{
			Path:      "api-port",
			Env:       "",
			Argument:  "api-port",
			Shorthand: "p",
			Default:   8080,
			Usage:     "Sensu Go Backend API Port (e.g. 4242)",
			Value:     &plugin.APIPort,
		},
		&sensu.PluginConfigOption{
			Path:      "api-user",
			Env:       "SENSU_API_USER",
			Argument:  "api-user",
			Shorthand: "u",
			Default:   "admin",
			Usage:     "Sensu Go Backend API User",
			Value:     &plugin.APIUser,
		},
		&sensu.PluginConfigOption{
			Path:      "api-pass",
			Env:       "SENSU_API_PASSWORD",
			Argument:  "api-pass",
			Shorthand: "P",
			Default:   "P@ssw0rd!",
			Usage:     "Sensu Go Backend API Password",
			Value:     &plugin.APIPass,
		},
		&sensu.PluginConfigOption{
			Path:      "api-key",
			Env:       "SENSU_API_KEY",
			Argument:  "api-key",
			Shorthand: "k",
			Default:   "",
			Usage:     "Sensu Go Backend API Key",
			Value:     &plugin.APIKey,
		},
		&sensu.PluginConfigOption{
			Path:      "warn-percent",
			Env:       "",
			Argument:  "warn-percent",
			Shorthand: "w",
			Default:   0,
			Usage:     "Warning threshold - % of Events in warning state",
			Value:     &plugin.WarnPercent,
		},
		&sensu.PluginConfigOption{
			Path:      "crit-percent",
			Env:       "",
			Argument:  "crit-percent",
			Shorthand: "c",
			Default:   0,
			Usage:     "Critical threshold - % of Events in warning state",
			Value:     &plugin.CritPercent,
		},
		&sensu.PluginConfigOption{
			Path:      "warn-count",
			Env:       "",
			Argument:  "warn-count",
			Shorthand: "W",
			Default:   0,
			Usage:     "Warning threshold - count of Events in warning state",
			Value:     &plugin.WarnCount,
		},
		&sensu.PluginConfigOption{
			Path:      "crit-count",
			Env:       "",
			Argument:  "crit-count",
			Shorthand: "C",
			Default:   0,
			Usage:     "Critical threshold - count of Events in warning state",
			Value:     &plugin.CritCount,
		},
		&sensu.PluginConfigOption{
			Path:      "secure",
			Env:       "",
			Argument:  "secure",
			Shorthand: "s",
			Default:   false,
			Usage:     "Use TLS connection to API",
			Value:     &plugin.Secure,
		},
		&sensu.PluginConfigOption{
			Path:      "insecure-skip-verify",
			Env:       "",
			Argument:  "insecure-skip-verify",
			Shorthand: "i",
			Default:   false,
			Usage:     "skip TLS certificate verification (not recommended!)",
			Value:     &plugin.InsecureSkipVerify,
		},
		&sensu.PluginConfigOption{
			Path:      "trusted-ca-file",
			Env:       "",
			Argument:  "trusted-ca-file",
			Shorthand: "t",
			Default:   "",
			Usage:     "TLS CA certificate bundle in PEM format",
			Value:     &plugin.TrustedCAFile,
		},
	}
)

func main() {
	check := sensu.NewGoCheck(&plugin.PluginConfig, options, checkArgs, executeCheck, false)
	check.Execute()
}

func checkArgs(event *types.Event) (int, error) {
	if len(plugin.CheckLabels) == 0 {
		return sensu.CheckStateWarning, fmt.Errorf("--check-labels is required")
	}
	if plugin.Secure {
		plugin.Protocol = "https"
	} else {
		plugin.Protocol = "http"
	}
	if len(plugin.TrustedCAFile) > 0 {
		caCertPool, err := corev2.LoadCACerts(plugin.TrustedCAFile)
		if err != nil {
			return sensu.CheckStateWarning, fmt.Errorf("Error loading specified CA file")
		}
		tlsConfig.RootCAs = caCertPool
	}
	tlsConfig.InsecureSkipVerify = plugin.InsecureSkipVerify

	tlsConfig.BuildNameToCertificate()
	tlsConfig.CipherSuites = corev2.DefaultCipherSuites

	return sensu.CheckStateOK, nil
}

func authenticate() (Auth, error) {
	var auth Auth
	client := http.DefaultClient
	client.Transport = http.DefaultTransport

	if plugin.Secure {
		client.Transport.(*http.Transport).TLSClientConfig = &tlsConfig
	}

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s://%s:%d/auth", plugin.Protocol, plugin.APIHost, plugin.APIPort),
		nil,
	)
	if err != nil {
		return auth, err
	}

	req.SetBasicAuth(plugin.APIUser, plugin.APIPass)

	resp, err := client.Do(req)
	if err != nil {
		return auth, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return auth, err
	}

	err = json.NewDecoder(bytes.NewReader(body)).Decode(&auth)

	return auth, err
}

func parseLabelArg(labelArg string) map[string]string {
	labels := map[string]string{}

	pairs := strings.Split(labelArg, ",")

	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}

	return labels
}

func filterEvents(events []*types.Event) []*types.Event {
	result := []*types.Event{}

	cLabels := parseLabelArg(plugin.CheckLabels)
	eLabels := parseLabelArg(plugin.EntityLabels)

	for _, event := range events {
		selected := true

		for key, value := range cLabels {
			if event.Check.ObjectMeta.Labels[key] != value {
				selected = false
				break
			}
		}

		if selected {
			for key, value := range eLabels {
				if event.Entity.ObjectMeta.Labels[key] != value {
					selected = false
					break
				}
			}
		}

		if selected {
			result = append(result, event)
		}
	}

	return result
}

func getEvents(auth Auth, namespace string) ([]*types.Event, error) {
	client := http.DefaultClient
	client.Transport = http.DefaultTransport

	url := fmt.Sprintf("%s://%s:%d/api/core/v2/namespaces/%s/events", plugin.Protocol, plugin.APIHost, plugin.APIPort, namespace)
	events := []*types.Event{}

	if plugin.Secure {
		client.Transport.(*http.Transport).TLSClientConfig = &tlsConfig
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return events, err
	}

	if len(plugin.APIKey) == 0 {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", auth.AccessToken))
	} else {
		req.Header.Set("Authorization", fmt.Sprintf("Key %s", plugin.APIKey))
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return events, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events, err
	}

	err = json.Unmarshal(body, &events)
	if err != nil {
		return events, err
	}

	result := filterEvents(events)

	return result, err
}

func executeCheck(event *types.Event) (int, error) {
	var autherr error
	auth := Auth{}

	if len(plugin.APIKey) == 0 {
		auth, autherr = authenticate()

		if autherr != nil {
			return sensu.CheckStateUnknown, autherr
		}
	}

	events := []*types.Event{}

	for _, namespace := range strings.Split(plugin.Namespaces, ",") {
		selected, err := getEvents(auth, namespace)

		if err != nil {
			return sensu.CheckStateUnknown, err
		}

		for _, event := range selected {
			events = append(events, event)
		}
	}

	counters := Counters{}

	entities := map[string]string{}
	checks := map[string]string{}

	for _, event := range events {
		entities[event.Entity.ObjectMeta.Name] = ""
		checks[event.Check.ObjectMeta.Name] = ""

		switch event.Check.Status {
		case 0:
			counters.Ok++
		case 1:
			counters.Warning++
		case 2:
			counters.Critical++
		default:
			counters.Unknown++
		}

		counters.Total++
	}

	counters.Entities = len(entities)
	counters.Checks = len(checks)

	fmt.Printf("Counters: %+v\n", counters)

	if counters.Total == 0 {
		fmt.Printf("WARNING: No Events returned for Aggregate\n")
		return sensu.CheckStateWarning, nil
	}

	percent := int((float64(counters.Ok) / float64(counters.Total)) * 100)

	fmt.Printf("Percent OK: %v\n", percent)

	if plugin.CritPercent != 0 {
		if percent <= plugin.CritPercent {
			fmt.Printf("CRITICAL: Less than %d%% percent OK (%d%%)\n", plugin.CritPercent, percent)
			return sensu.CheckStateCritical, nil
		}
	}

	if plugin.WarnPercent != 0 {
		if percent <= plugin.WarnPercent {
			fmt.Printf("WARNING: Less than %d%% percent OK (%d%%)\n", plugin.WarnPercent, percent)
			return sensu.CheckStateWarning, nil
		}
	}

	if plugin.CritCount != 0 {
		if counters.Critical >= plugin.CritCount {
			fmt.Printf("CRITICAL: %d or more Events are in a Critical state (%d)\n", plugin.CritCount, counters.Critical)
			return sensu.CheckStateCritical, nil
		}
	}

	if plugin.WarnCount != 0 {
		if counters.Warning >= plugin.WarnCount {
			fmt.Printf("WARNING: %d or more Events are in a Warning state (%d)\n", plugin.WarnCount, counters.Warning)
			return sensu.CheckStateWarning, nil
		}
	}

	fmt.Printf("Everything is OK\n")

	return sensu.CheckStateOK, nil
}
