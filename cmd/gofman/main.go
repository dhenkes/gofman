package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/dhenkes/gofman/pkg/auth"
	"github.com/dhenkes/gofman/pkg/gofman"
	"github.com/dhenkes/gofman/pkg/http"
	"github.com/dhenkes/gofman/pkg/path_traversal"
	"github.com/dhenkes/gofman/pkg/sqlite"
	"github.com/pelletier/go-toml"
)

// Build version, injected during build.
var (
	version string
	commit  string
)

// Default settings.
const (
	DefaultConfigPath  = "~/.gofman/config.toml"
	DefaultDatabaseDSN = "~/.gofman/db"
	DefaultHTTPAddress = "127.0.0.1"
	DefaultHTTPPort    = 8080
)

func main() {
	gofman.Version = strings.TrimPrefix(version, "")
	gofman.Commit = commit

	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() { <-c; cancel() }()

	m := NewMain()

	m.DB.AuthService = m.AuthService

	fs := flag.NewFlagSet("gofman", flag.ContinueOnError)
	fs.StringVar(&m.ConfigPath, "config", DefaultConfigPath, "config path")

	if err := fs.Parse(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	configPath, err := m.PathTraversalService.Expand(m.ConfigPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	buf, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = toml.Unmarshal(buf, &m.Config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := m.Run(ctx); err != nil {
		m.Close()
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	<-ctx.Done()

	if err := m.Close(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Main represents the program.
type Main struct {
	Config     Config
	ConfigPath string

	DB *sqlite.DB

	HTTPServer *http.Server

	AuthService          gofman.AuthService
	PathTraversalService gofman.PathTraversalService
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		Config:     NewConfig(),
		ConfigPath: DefaultConfigPath,

		DB: sqlite.NewDB(),

		HTTPServer: http.NewServer(),

		AuthService:          auth.NewAuthService(),
		PathTraversalService: path_traversal.NewPathTraversalService(),
	}
}

// Config represents the CLI configuration file.
type Config struct {
	HTTP struct {
		Address string `toml:"address"`
		Port    int    `toml:"port"`
	} `toml:"http"`

	Database struct {
		DSN string `toml:"dsn"`
	} `toml:"database"`
}

// NewConfig returns a new instance of Config with defaults set.
func NewConfig() Config {
	var config Config

	config.Database.DSN = DefaultDatabaseDSN

	config.HTTP.Address = DefaultHTTPAddress
	config.HTTP.Port = DefaultHTTPPort

	return config
}

// Close gracefully stops the program.
func (m *Main) Close() error {
	if m.HTTPServer != nil {
		if err := m.HTTPServer.Close(); err != nil {
			return err
		}
	}

	if m.DB != nil {
		if err := m.DB.Close(); err != nil {
			return err
		}
	}

	return nil
}

// Run executes the program. The configuration should already be set up before
// calling this function.
func (m *Main) Run(ctx context.Context) (err error) {
	if m.DB.DSN, err = m.PathTraversalService.Expand(m.Config.Database.DSN); err != nil {
		return err
	}

	if err := m.DB.Open(); err != nil {
		return err
	}

	m.HTTPServer.Address = m.Config.HTTP.Address
	m.HTTPServer.Port = m.Config.HTTP.Port

	m.HTTPServer.ActorService = sqlite.NewActorService(m.DB)
	m.HTTPServer.FileService = sqlite.NewFileService(m.DB)
	m.HTTPServer.SessionService = sqlite.NewSessionService(m.DB)
	m.HTTPServer.SetupService = sqlite.NewSetupService(m.DB)
	m.HTTPServer.TagService = sqlite.NewTagService(m.DB)
	m.HTTPServer.UserService = sqlite.NewUserService(m.DB)
	m.HTTPServer.AuthService = m.AuthService
	m.HTTPServer.PathTraversalService = m.PathTraversalService

	if err := m.HTTPServer.Open(); err != nil {
		return err
	}

	log.Printf("Running: url=%q dsn=%q", m.HTTPServer.URL(), m.Config.Database.DSN)

	return nil
}
