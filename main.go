package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/sensu/sensu-go/types"
	"github.com/spf13/cobra"
)

var (
	checkLabels  string
	entityLabels string
	namespaces   string
	apiURL       string
	apiPort      string
	apiUser      string
	apiPass      string
	trustedCA    string
	warnPercent  int
	critPercent  int
	warnCount    int
	critCount    int
	debug        int
)

// Auth struct
type Auth struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// Counters struct
type Counters struct {
	Entities int
	Checks   int
	Ok       int
	Warning  int
	Critical int
	Unknown  int
	Total    int
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	rootCmd := configureRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func configureRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sensu-aggregate-check",
		Short: "The Sensu Go Event Aggregates Check plugin",
		RunE:  run,
	}

	cmd.Flags().StringVarP(&checkLabels,
		"check-labels",
		"l",
		"",
		"Sensu Go Event Check Labels to filter by (e.g. 'aggregate=foo')")

	cmd.Flags().StringVarP(&entityLabels,
		"entity-labels",
		"e",
		"",
		"Sensu Go Event Entity Labels to filter by (e.g. 'aggregate=foo,app=bar')")

	cmd.Flags().StringVarP(&namespaces,
		"namespaces",
		"n",
		"default",
		"Comma-delimited list of Sensu Go Namespaces to query for Events (e.g. 'us-east-1,us-west-2')")

	cmd.Flags().StringVarP(&trustedCA,
		"ca",
		"k",
		"",
		"Trusted CA file")

	cmd.Flags().StringVarP(&apiURL,
		"api-host",
		"H",
		"http://127.0.0.1",
		"Sensu Go Backend API Host (e.g. 'https://sensu-backend.example.com', http://127.0.0.1)")

	cmd.Flags().StringVarP(&apiPort,
		"api-port",
		"p",
		"8080",
		"Sensu Go Backend API Port (e.g. 4242)")

	cmd.Flags().StringVarP(&apiUser,
		"api-user",
		"u",
		"admin",
		"Sensu Go Backend API User")

	cmd.Flags().StringVarP(&apiPass,
		"api-pass",
		"P",
		"P@ssw0rd!",
		"Sensu Go Backend API User")

	cmd.Flags().IntVarP(&warnPercent,
		"warn-percent",
		"w",
		0,
		"Warning threshold - % of Events in warning state")

	cmd.Flags().IntVarP(&critPercent,
		"crit-percent",
		"c",
		0,
		"Critical threshold - % of Events in critical state")

	cmd.Flags().IntVarP(&warnCount,
		"warn-count",
		"W",
		0,
		"Warning threshold - count of Events in warning state")

	cmd.Flags().IntVarP(&critCount,
		"crit-count",
		"C",
		0,
		"Critical threshold - count of Events in critical state")

	cmd.Flags().IntVarP(&debug,
		"debug",
		"d",
		0,
		"Spam terminal - lvl 0-1")

	_ = cmd.MarkFlagRequired("check-labels")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		_ = cmd.Help()
		return fmt.Errorf("invalid argument(s) received")
	}

	return evalAggregate()
}

func clientHTTP() (client *http.Client) {
	var urlHTTPSScheme = regexp.MustCompile(`^https://.+`)
	// https url specified
	if urlHTTPSScheme.MatchString(apiURL) {

		caCertPool, err := x509.SystemCertPool()
		if err != nil {
			log.Fatal(err)
		}
		if caCertPool == nil {
			caCertPool = x509.NewCertPool()
		}
		// custom ACs given
		if len(trustedCA) > 0 {
			caCert, err := ioutil.ReadFile(fmt.Sprintf("%s", trustedCA))
			if err != nil {
				log.Fatal(err)
			}
			caCertPool.AppendCertsFromPEM(caCert)
		}
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					RootCAs:            caCertPool,
					InsecureSkipVerify: false,
				},
			},
		}
		return client
	}
	// plain http
	return &http.Client{}
}

func authenticate() (Auth, error) {
	var auth Auth
	client := clientHTTP()
	url := fmt.Sprintf("%s:%s/auth", apiURL, apiPort)
	if debug >= 0 {
		log.Println(fmt.Sprintf("authenticate URL: %s", url))
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Couldn't build request")
		return auth, err
	}

	req.SetBasicAuth(apiUser, apiPass)

	resp, err := client.Do(req)
	if err != nil {
		log.Println("Couldn't perform request")
		return auth, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Failed to read response body")
		return auth, err
	}

	err = json.NewDecoder(bytes.NewReader(body)).Decode(&auth)
	if err != nil {
		log.Println(body)
		log.Fatal(err)
	}

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

	cLabels := parseLabelArg(checkLabels)
	eLabels := parseLabelArg(entityLabels)

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
	client := clientHTTP()

	url := fmt.Sprintf("%s:%s/api/core/v2/namespaces/%s/events", apiURL, apiPort, namespace)
	events := []*types.Event{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return events, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", auth.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	if debug >= 0 {
		log.Println(fmt.Sprintf("getEvents req: %+v", req))
	}
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

	if debug >= 1 {
		log.Println(fmt.Sprintf("getEvents resp body: %s", string(body)))
	}

	result := filterEvents(events)

	return result, err
}

func evalAggregate() error {
	auth, err := authenticate()

	if err != nil {
		return err
	}

	events := []*types.Event{}

	for _, namespace := range strings.Split(namespaces, ",") {
		selected, err := getEvents(auth, namespace)

		if err != nil {
			return err
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
		os.Exit(1)
	}

	percent := int((float64(counters.Ok) / float64(counters.Total)) * 100)

	fmt.Printf("Percent OK: %v\n", percent)

	if critPercent != 0 {
		if percent <= critPercent {
			fmt.Printf("CRITICAL: Less than %d%% percent OK (%d%%)\n", critPercent, percent)
			os.Exit(2)
		}
	}

	if warnPercent != 0 {
		if percent <= warnPercent {
			fmt.Printf("WARNING: Less than %d%% percent OK (%d%%)\n", warnPercent, percent)
			os.Exit(1)
		}
	}

	if critCount != 0 {
		if counters.Critical >= critCount {
			fmt.Printf("CRITICAL: %d or more Events are in a Critical state (%d)\n", critCount, counters.Critical)
			os.Exit(2)
		}
	}

	if warnCount != 0 {
		if counters.Warning >= warnCount {
			fmt.Printf("WARNING: %d or more Events are in a Warning state (%d)\n", warnCount, counters.Warning)
			os.Exit(2)
		}
	}

	fmt.Printf("Everything is OK\n")

	return err
}
