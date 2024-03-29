/*
Copyright 2020–2021 MET Norway

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/metno/go-mms/internal/server"
	"github.com/metno/go-mms/pkg/gencert"
	"github.com/metno/go-mms/pkg/mms"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sethvargo/go-password/password"

	nats "github.com/nats-io/nats-server/v2/server"
	natscli "github.com/nats-io/nats.go"

	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const productionHubName = "default"
const confFile = "mmsd_config.yml"
const dbEventsFile = "events.db"
const dbStateFile = "state.db"
const dbJWTFile = "jwt.db"

func main() {

	var err error
	var hubID string

	// Create an identifier
	hubID, err = mms.MakeHubIdentifier()
	if err != nil {
		log.Printf("Failed to create identifier, %s", err.Error())
		hubID = "error"
	}

	cmdFlags := []cli.Flag{
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "work-dir",
			Aliases: []string{"w"},
			Value:   ".",
			Usage:   "The working directory where the files for this instance are stored.",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "hubid",
			Usage: "Production hub identifier. If not specified, an identifier is generated.",
			Value: hubID,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "hostname",
			Usage: "Specify the hostname for API and NATS.",
			Value: "localhost",
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:  "api-port",
			Usage: "Specify the port number for the API listening port.",
			Value: 8080,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:  "nats-port",
			Usage: "Specify the port number for the NATS listening port.",
			Value: 4222,
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:  "nats-local",
			Usage: "Specify wether this MMSD instance will spawn own NATS-server (True) or connect to an existing NATS stream",
			Value: true,
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "nats-cred-path",
			Usage: "Path where creds are stored in for mmsd-sidecar",
			Value: "/nsc/nkeys/creds/met-operator/met-account/",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "nats-url",
			Usage: "Specify which nats-url daemon should post incoming messages",
			Value: "",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "heartbeat-user",
			Aliases: []string{"cred-file"},
			Usage:   "Username to use to post heartbeat to external NATS",
			Value:   "",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "heartbeat-password",
			Usage: "Password to use to post heartbeat to external NATS",
			Value: "",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:    "certificate",
			Aliases: []string{"cert"},
			Usage:   "Specify the path to the certificate.",
			Value:   "cert.pem",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "key",
			Usage: "Specify the path to the key.",
			Value: "key.pem",
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:  "tls",
			Usage: "Enable TLS",
			Value: false,
		}),
		altsrc.NewIntFlag(&cli.IntFlag{
			Name:  "heartbeat-interval",
			Usage: "Specify the interval for sending heartbeats. Turn off with 0 or negative value",
			Value: 10,
		}),
		altsrc.NewUintFlag(&cli.UintFlag{
			Name:  "del-events-interval",
			Usage: "Specify the interval(hours) for deleting events. Default is 12 hours (deletes 12 hours old events)",
			Value: 12,
		}),
	}

	certFlags := []cli.Flag{
		&cli.StringFlag{
			Name:  "common-name",
			Usage: "CN of the certificate.",
			Value: "localhost",
		},
		&cli.StringFlag{
			Name:  "alternative-names",
			Usage: "Comma separated list of Alternative Names for certificate generation",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "country",
			Usage: "Country abbreviation for the issued certificate.",
			Value: "NO",
		},
		&cli.BoolFlag{
			Name:  "overwrite",
			Usage: "If key.pem exists from before, it won't be overwritten unless --overwrite is specified.",
			Value: false,
		},
	}

	app := &cli.App{
		Before: func(ctx *cli.Context) error {
			confPath := fmt.Sprint(filepath.Join(ctx.String("work-dir"), confFile))
			inputSource, err := altsrc.NewYamlSourceFromFile(confPath)
			if err != nil {
				// If there is no file, just return without error
				return nil
			}

			return altsrc.ApplyInputSourceValues(ctx, inputSource, cmdFlags)
		},
		Flags: cmdFlags,
		Action: func(ctx *cli.Context) error {
			var natsURL string
			// natsUser is either filepath for JWT or cred file, or privateUser for local nats
			var natsUser string
			// if natsUser is JWT, natsPassword is nkeySeed filepath, if local nats, it local pass
			var natsPassword string
			var natsCredentials natscli.Option

			var eventsDB *sql.DB
			var stateDB *sql.DB

			natsLocal := ctx.Bool("nats-local")

			if natsLocal {
				natsURL = fmt.Sprintf("nats://%s:%d", ctx.String("hostname"), ctx.Int("nats-port"))
				natsUser = "privateUser"
				// Create a password to use internally if NATS is local for privateUser
				natsPassword, err = password.Generate(64, 10, 10, false, false)
				if err != nil {
					log.Fatal(err)
				}
				privateNatsUser := &nats.User{
					Username: natsUser,
					Password: natsPassword,
					Permissions: &nats.Permissions{
						Publish: &nats.SubjectPermission{
							Allow: []string{"mms"},
						},
						Subscribe: &nats.SubjectPermission{
							Allow: []string{"mms"},
						},
					},
				}

				publicNatsUser := &nats.User{
					Username: "publicUser",
					Permissions: &nats.Permissions{
						Publish: &nats.SubjectPermission{
							Deny: []string{"*"},
						},
						Subscribe: &nats.SubjectPermission{
							Allow: []string{"mms"},
						},
					},
				}

				users := []*nats.User{privateNatsUser, publicNatsUser}

				opts := &nats.Options{
					ServerName: fmt.Sprintf("mmsd-nats-server-%s", productionHubName),
					Host:       ctx.String("hostname"),
					Port:       ctx.Int("nats-port"),
					Users:      users,
					NoAuthUser: "publicUser",
				}

				natsServer, err := nats.NewServer(opts)
				if err != nil {
					nats.PrintAndDie(fmt.Sprintf("nats server failed: %s for server: mmsd-nats-server-%s", err, productionHubName))
				}

				startNATSServer(natsServer, natsURL)
				natsCredentials = natscli.UserInfo("privateUser", natsPassword)
			} else {
				natsURL = ctx.String("nats-url")
				if natsURL == "" {
					return fmt.Errorf("Need to provide nats-url if nats-local is false")
				}
				natsUser = ctx.String("heartbeat-user")
				if natsUser == "" {
					return fmt.Errorf("Need to provide either JWT and NkeySeed or cred files as heartbeat-user")
				}
				natsPassword = ctx.String("heartbeat-password")
				if natsPassword == "" {
					natsCredentials = natscli.UserCredentials(natsUser)
				} else {
					natsCredentials = natscli.UserCredentials(natsUser, natsPassword)
				}
			}

			apiURL := fmt.Sprintf("%s:%d", ctx.String("hostname"), ctx.Int("api-port"))

			eventsPath := fmt.Sprint(filepath.Join(ctx.String("work-dir"), dbEventsFile))
			eventsDB, err = server.NewEventsDB(eventsPath)
			if err != nil {
				log.Fatalf("could not open events db: %s", err)
			}
			if natsLocal {
				statePath := fmt.Sprint(filepath.Join(ctx.String("work-dir"), dbStateFile))
				stateDB, err = server.NewStateDB(statePath)
				if err != nil {
					log.Fatalf("could not open state db for local NATS authentication: %s", err)
				}
			} else {
				statePath := fmt.Sprint(filepath.Join(ctx.String("work-dir"), dbJWTFile))
				NSC_creds_location := ctx.String("nats-cred-path")
				stateDB, err = server.NewJWTDB(statePath, NSC_creds_location)
				if err != nil {
					log.Fatalf("could not open state db for non-local NATS authentication: %s", err)
				}
			}

			templates := server.CreateTemplates()

			webService := server.NewService(templates, eventsDB, stateDB, natsURL, natsCredentials, server.Version{Version: version, Commit: commit, Date: date}, natsLocal)

			log.Println("Populating productstatus from the local events database ...")
			events, err := webService.GetAllEvents(context.Background())
			if err != nil {
				log.Fatalf("could not read all events %s", err)
			}
			webService.Productstatus.Populate(events)

			if natsLocal {

				heartBeatInterval := ctx.Int("heartbeat-interval")

				if heartBeatInterval > 0 {
					startHeartBeat(heartBeatInterval, natsURL, natsCredentials, natsLocal)
				}
			}
			startEventLoop(webService, ctx.Int(("del-events-interval")))
			startWebServer(webService, apiURL, ctx.Bool("tls"), ctx.String("certificate"), ctx.String("key"))

			return nil
		},
		Commands: []*cli.Command{
			{
				Name:  "keys",
				Usage: fmt.Sprintf("Manage API keys."),
				Flags: []cli.Flag{
					altsrc.NewBoolFlag(&cli.BoolFlag{
						Name:    "gen",
						Aliases: []string{"g"},
						Usage:   "Generate a new API key and add it to the autorized keys.",
					}),
					altsrc.NewBoolFlag(&cli.BoolFlag{
						Name:    "list",
						Aliases: []string{"l"},
						Usage:   "List all keys in autorized keys.",
					}),
					altsrc.NewStringFlag(&cli.StringFlag{
						Name:    "add",
						Aliases: []string{"a"},
						Usage:   "Add a new API key to autorized keys.",
						Value:   "None",
					}),
					altsrc.NewStringFlag(&cli.StringFlag{
						Name:    "remove",
						Aliases: []string{"r"},
						Usage:   "Remove an API key from autorized keys.",
						Value:   "None",
					}),
					altsrc.NewStringFlag(&cli.StringFlag{
						Name:    "message",
						Aliases: []string{"m"},
						Usage:   "A descriptive message for the generated or added key.",
						Value:   "Unnamed key",
					}),
				},
				Action: func(ctx *cli.Context) error {
					// Open the database
					statePath := fmt.Sprint(filepath.Join(ctx.String("work-dir"), dbStateFile))
					stateDB, err := server.NewStateDB(statePath)
					if err != nil {
						log.Fatalf("could not open state db: %s", err)
					}

					if ctx.Bool("gen") {
						err := generateAPIKey(stateDB, ctx.String("message"))
						if err != nil {
							log.Fatalf("failed to generate key: %s", err)
						}
					} else if ctx.Bool("list") {
						err := server.ListApiKeys(stateDB)
						if err != nil {
							log.Fatalf("failed to list keys: %s", err)
						}
					} else if ctx.String("add") != "None" {
						err := server.AddNewApiKey(stateDB, ctx.String("add"), ctx.String("message"))
						if err != nil {
							log.Fatalf("failed to add key: %s", err)
						}
						fmt.Printf("Added Key:   %s\n", ctx.String("add"))
						fmt.Printf("Key Message: %s\n", ctx.String("message"))
					} else if ctx.String("remove") != "None" {
						isOk, err := server.RemoveApiKey(stateDB, ctx.String("remove"))
						if err != nil {
							log.Fatalf("failed to remove key: %s", err)
						}
						if isOk {
							fmt.Printf("Removed Key: %s\n", ctx.String("remove"))
						} else {
							fmt.Printf("Key Not Found: %s\n", ctx.String("remove"))
						}
					} else {
						fmt.Println("No action selected, please refer to help by using 'keys --help' or 'keys -h'")
					}

					return nil
				},
			},
			{
				Name:    "generate-certificate",
				Aliases: []string{"gencert"},
				Usage:   "Generate a private key (key.pem) and X509 certificate (cert.pem).",
				Flags:   certFlags,
				Action:  gencert.GenerateCertificate(),
			},
			{
				Name:    "generate-csr",
				Aliases: []string{"gencsr"},
				Usage:   "Generate a private key (key.pem) and a signing request (cert.csr).",
				Flags:   certFlags,
				Action:  gencert.GenerateCSR(),
			},
			{
				Name:    "version",
				Aliases: []string{"v"},
				Usage:   "Show build information",
				Action: func(ctx *cli.Context) error {
					fmt.Printf("Version: %s\nCommit: %s\nBuilt on: %s\n", version, commit, date)
					return nil
				},
			},
		},
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func startNATSServer(natsServer *nats.Server, natsURL string) {
	go func() {
		log.Printf("Starting NATS server on %s ...", natsURL)
		if err := nats.Run(natsServer); err != nil {
			nats.PrintAndDie(err.Error())
		}
		natsServer.WaitForShutdown()
	}()
}

func startHeartBeat(heartBeatInterval int, natsURL string, natsCredentials natscli.Option, natsLocal bool) {

	var pEvent mms.HeartBeatEvent
	log.Printf("Starting heartbeat sender with interval: %d s", heartBeatInterval)

	interval := time.Duration(heartBeatInterval)
	ticker := time.NewTicker(interval * time.Second)

	pEvent = mms.HeartBeatEvent{
		ProductionHub: "heartBeat",
	}

	go func() {
		for {
			select {
			case <-ticker.C:
				pEvent.CreatedAt = time.Now()
				pEvent.NextEventAt = time.Now().Add(interval)
				if err := mms.MakeHeartBeatEvent(natsURL, natsCredentials, &pEvent, natsLocal); err != nil {
					log.Printf("failed to send HeartBeat message: %s", err.Error())
				}
			}
		}
	}()
}

func startEventLoop(webService *server.Service, eventDeletionInterval int) {
	log.Printf("Starting event loop with %v hours of event deletion Interval ...", eventDeletionInterval)
	// Start a separate go routine serving as an event loop for maintenance tasks.

	uptimeCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Subsystem: "mmsd",
		Name:      "uptime_seconds_total",
		Help:      "The total number of seconds since the start of the application.",
	})

	webService.Metrics.MustRegister(uptimeCounter)

	secondTicker := time.NewTicker(1 * time.Second)
	go func() {
		for {
			select {
			case <-secondTicker.C:
				uptimeCounter.Inc()
				webService.Productstatus.UpdateMetrics()
			}
		}
	}()

	hourTicker := time.NewTicker(1 * time.Hour)
	go func() {
		for {
			select {
			case <-hourTicker.C:
				if err := webService.DeleteOldEvents(time.Now().Add(-time.Hour * time.Duration(eventDeletionInterval))); err != nil {
					log.Printf("failed to delete old events from events db: %s", err)
				}
			}
		}
	}()
}

func startWebServer(webService *server.Service, apiURL string, tlsEnabled bool, certificatePath string, keyPath string) {
	server := &http.Server{
		Addr:         apiURL,
		Handler:      webService.Router,
		WriteTimeout: 1 * time.Second,
		IdleTimeout:  10 * time.Second,
	}
	log.Printf("Starting webserver on %s ...\n", server.Addr)
	if tlsEnabled {
		log.Fatal(server.ListenAndServeTLS(certificatePath, keyPath))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}

func generateAPIKey(stateDB *sql.DB, keyMsg string) error {
	// Seeding the random generator for each call may be risky since it may produce the same
	// seed twice if the time resolution is low and the function is called often. However, the
	// function is only called once in a single instance of mmsd, and the database should error
	// on a duplicate key entry.
	rand.Seed(time.Now().UnixNano())

	// Generate the key
	byteKey := make([]byte, 32)
	for i := range byteKey {
		byteKey[i] = byte(rand.Intn(255))
	}
	apiKey := base64.StdEncoding.EncodeToString([]byte(byteKey))

	// Save the new key entry
	err := server.AddNewApiKey(stateDB, apiKey, keyMsg)
	if err != nil {
		log.Fatalf("error in state db: %s", err)
	}

	fmt.Printf("Generated Key: %s\n", apiKey)
	fmt.Printf("Key Message:   %s\n", keyMsg)

	return nil
}
