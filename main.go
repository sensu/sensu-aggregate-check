package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/sensu/sensu-go/types"
	"github.com/spf13/cobra"
)

var (
	labels     string
	namespaces string
	apiHost    string
	apiPort    string
	apiUser    string
	apiPass    string
)

type Auth struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

func main() {
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

	cmd.Flags().StringVarP(&labels,
		"labels",
		"l",
		"",
		"aggregate=foo,app=bar")

	cmd.Flags().StringVarP(&namespaces,
		"namespaces",
		"n",
		"default",
		"us-east-1,us-west-2")

	cmd.Flags().StringVarP(&apiHost,
		"api-host",
		"H",
		"127.0.0.1",
		"sensu-backend.example.com")

	cmd.Flags().StringVarP(&apiPort,
		"api-port",
		"p",
		"8080",
		"5555")

	cmd.Flags().StringVarP(&apiUser,
		"api-user",
		"u",
		"admin",
		"ackbar")

	cmd.Flags().StringVarP(&apiPass,
		"api-pass",
		"P",
		"P@ssw0rd!",
		"itsatrap")

	_ = cmd.MarkFlagRequired("labels")

	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) != 0 {
		_ = cmd.Help()
		return fmt.Errorf("invalid argument(s) received")
	}

	return evalAggregate()
}

func authenticate() (Auth, error) {
	var auth Auth
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("http://%s:%s/auth", apiHost, apiPort),
		nil,
	)
	if err != nil {
		return auth, err
	}

	req.SetBasicAuth(apiUser, apiPass)

	resp, err := http.DefaultClient.Do(req)
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

func getEvents(auth Auth, namespace string, labels string) ([]*types.Event, error) {
	url := fmt.Sprintf("http://%s:%s/api/core/v2/namespaces/%s/events", apiHost, apiPort, namespace)
	events := []*types.Event{}
	req, err := http.NewRequest(
		"GET",
		url,
		nil,
	)

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", auth.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return events, err
	}

	defer resp.Body.Close()
	b, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(b))

	return events, nil
}

func evalAggregate() error {
	auth, err := authenticate()

	if err != nil {
		return err
	}

	events, err := getEvents(auth, "default", "aggregate=foo")

	fmt.Printf("hello world: %s\n", events)

	return err
}
