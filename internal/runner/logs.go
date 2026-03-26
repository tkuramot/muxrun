package runner

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tkuramot/muxrun/internal/tmux"
)

type LogsOptions struct {
	GroupName string
	AppName   string
	Follow    bool
	Writer    io.Writer
}

func (r *Runner) Logs(ctx context.Context, opts LogsOptions) error {
	groups := r.cfg.FindGroups(opts.GroupName)
	if len(groups) == 0 {
		return fmt.Errorf("%w: %s", ErrGroupNotFound, opts.GroupName)
	}
	g := groups[0]

	app := g.FindApp(opts.AppName)
	if app == nil {
		return fmt.Errorf("%w: %s in group %s", ErrAppNotFound, opts.AppName, g.Name)
	}

	session := tmux.SessionName(g.Name)

	if opts.Follow {
		return r.followLogs(ctx, session, app.Name, opts.Writer)
	}

	hasWin, err := tmux.HasWindow(r.tmux, session, app.Name)
	if err != nil {
		return err
	}
	if !hasWin {
		return nil
	}

	output, err := r.tmux.CapturePane(session, app.Name)
	if err != nil {
		return err
	}
	fmt.Fprint(opts.Writer, output)
	return nil
}

func (r *Runner) followLogs(ctx context.Context, session, appName string, w io.Writer) error {
	tmpDir, err := os.MkdirTemp("", "muxrun-logs-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	logPath := filepath.Join(tmpDir, session+"-"+appName+".log")
	f, err := os.Create(logPath)
	if err != nil {
		return err
	}
	f.Close()

	if err := r.tmux.PipePane(session, appName, fmt.Sprintf("cat >> '%s'", logPath)); err != nil {
		return err
	}
	defer r.tmux.UnpipePane(session, appName) //nolint:errcheck

	existing, err := r.tmux.CapturePane(session, appName)
	if err != nil {
		return err
	}
	// tmux capture-pane outputs the full pane grid including empty trailing lines,
	// so the result always ends with one or more newlines. Trim them to avoid a
	// blank line between the existing output and the streamed lines that follow.
	existing = strings.TrimRight(existing, "\n")
	if existing != "" {
		fmt.Fprintln(w, existing)
	}

	f, err = os.Open(logPath)
	if err != nil {
		return err
	}
	defer f.Close()

	buf := bufio.NewReader(f)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		line, err := buf.ReadString('\n')
		if len(line) > 0 {
			fmt.Fprintln(w, strings.TrimRight(line, "\n"))
		}
		if err == io.EOF {
			time.Sleep(100 * time.Millisecond)
		} else if err != nil {
			return err
		}
	}
}
