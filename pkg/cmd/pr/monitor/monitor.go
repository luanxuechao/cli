package checks

import (
	"errors"
	"net/http"
	"time"

	"github.com/cli/cli/api"
	"github.com/cli/cli/context"
	"github.com/cli/cli/internal/ghrepo"
	"github.com/cli/cli/pkg/cmd/pr/shared"
	"github.com/cli/cli/pkg/cmdutil"
	"github.com/cli/cli/pkg/iostreams"
	gosxnotifier "github.com/deckarep/gosx-notifier"
	"github.com/spf13/cobra"
)

type MonitorOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	BaseRepo   func() (ghrepo.Interface, error)
	Branch     func() (string, error)
	Remotes    func() (context.Remotes, error)

	WebMode bool

	SelectorArg string
}

func NewCmdMonitor(f *cmdutil.Factory, runF func(*MonitorOptions) error) *cobra.Command {
	opts := &MonitorOptions{
		IO:         f.IOStreams,
		HttpClient: f.HttpClient,
		Branch:     f.Branch,
		Remotes:    f.Remotes,
		BaseRepo:   f.BaseRepo,
	}

	cmd := &cobra.Command{
		Use:   "monitor [<number> | <url> | <branch>]",
		Short: "monitor CI status for a single pull request",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// support `-R, --repo` override
			opts.BaseRepo = f.BaseRepo

			if repoOverride, _ := cmd.Flags().GetString("repo"); repoOverride != "" && len(args) == 0 {
				return &cmdutil.FlagError{Err: errors.New("argument required when using the --repo flag")}
			}

			if len(args) > 0 {
				opts.SelectorArg = args[0]
			}

			if runF != nil {
				return runF(opts)
			}

			return monitorRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.WebMode, "web", "w", false, "Open the web browser to show details about checks")

	return cmd
}

func monitorRun(opts *MonitorOptions) error {
	httpClient, err := opts.HttpClient()
	if err != nil {
		return err
	}
	apiClient := api.NewClientFromHTTP(httpClient)

	for {
		pr, _, err := shared.PRFromArgs(apiClient, opts.BaseRepo, opts.Branch, opts.Remotes, opts.SelectorArg)
		if err != nil {
			return err
		}
		checks := pr.BuildFinished()

		if checks {
			note := gosxnotifier.NewNotification("Check your PR status!")

			note.Title = "Build finished"

			note.Link = opts.SelectorArg

			note.Sound = gosxnotifier.Default
			err := note.Push()
			if err != nil {
				return err
			}
			break
		}
		time.Sleep(60 * time.Second)
	}

	return nil
}
