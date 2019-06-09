package main

import (
	"log"
	"os"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

const envToken = "TG_BOT_TOKEN"
const envPollTimeout = "POLL_TIMEOUT"
const envListen = "LISTEN_IP"
const envWebhook = "WEBHOOK"
const envWebhookPort = "PORT"
const envCert = "CERT"
const envKey = "CERT_KEY"
const envDebug = "DEBUG"

const defaultPollTimeout = 30
const defaultWebhookPort = 8000

const ModePolling = "polling"
const ModeWebhook = "webhook"

type BotConfig struct {
	Token           string
	Mode            string
	Debug           bool
	PollingTimeout  int
	WebhookURL      string
	ListenAddr      string
	CertificateFile string
	CertificateKey  string
}

func LoadConfig() (BotConfig, error) {
	var (
		cfg                      BotConfig
		pollTimeout, webhookPort int
	)

	if err := godotenv.Load(); err != nil {
		dir, err := filepath.Abs(filepath.Dir(os.Args[0]))

		if err != nil {
			log.Print(err)
		}

		if err := godotenv.Load(filepath.Join(filepath.Dir(dir), ".env")); err != nil {
			log.Print(err)
		}
	}

	token := os.Getenv(envToken)
	webhookUrl := os.Getenv(envWebhook)
	listenAddr := os.Getenv(envListen)
	webhookPortStr := os.Getenv(envWebhookPort)
	certFile := os.Getenv(envCert)
	certKey := os.Getenv(envKey)
	pollTimeoutStr := os.Getenv(envPollTimeout)
	debug, err := strconv.ParseBool(os.Getenv(envDebug))

	if err != nil {
		debug = false
	}

	if pollTimeout, err = strconv.Atoi(pollTimeoutStr); err != nil {
		pollTimeout = defaultPollTimeout
	}

	if webhookPort, err = strconv.Atoi(webhookPortStr); err != nil && webhookUrl != "" {
		webhookPort = defaultWebhookPort
	}

	webhookLink := strings.ReplaceAll(webhookUrl, "{PORT}", strconv.Itoa(webhookPort))
	webhookLink = strings.ReplaceAll(webhookLink, "{TOKEN}", token)

	if token == "" {
		log.Fatalf(
			"`%s` is not found in environment - specify it in `.env` file or pass it while launching",
			envToken,
		)
	}

	cfg = BotConfig{
		Token:           token,
		Debug:           debug,
		PollingTimeout:  pollTimeout,
		WebhookURL:      webhookLink,
		ListenAddr:      fmt.Sprintf("%s:%d", listenAddr, webhookPort),
		CertificateFile: certFile,
		CertificateKey:  certKey,
	}

	if webhookUrl == "" {
		cfg.Mode = ModePolling
	} else {
		cfg.Mode = ModeWebhook
	}

	return cfg, nil
}
