//go:build windows

package cmds

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"

	"xeneoncc/internal/config"
)

const (
	serviceName    = "XeneonBridge"
	serviceDisplay = "Xeneon Claude Code Bridge"
	serviceDesc    = "Serves live Claude Code usage and limits to the Xeneon Edge widget on loopback."
)

type bridgeService struct{}

func (bridgeService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.StartPending}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errc := make(chan error, 1)
	go func() { errc <- ServeContext(ctx) }()
	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				s <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				s <- svc.Status{State: svc.StopPending}
				cancel()
				<-errc // wait for graceful shutdown
				return false, 0
			default:
				log.Printf("unexpected service control request: %d", c.Cmd)
			}
		case err := <-errc: // bridge exited on its own (fatal bind error)
			if err != nil {
				log.Printf("bridge exited: %v", err)
				return false, 1
			}
			return false, 0
		}
	}
}

// ServiceRun is the SCM entrypoint (`xeneon-bridge service run`). It must be
// started by the Service Control Manager; from a console svc.Run returns an error.
func ServiceRun() error {
	if err := setupServiceLog(); err != nil {
		return err
	}
	return svc.Run(serviceName, bridgeService{})
}

func setupServiceLog() error {
	dir, err := config.Dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "bridge.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	log.SetOutput(f) // stays open for the service lifetime
	return nil
}

func connectManager() (*mgr.Mgr, error) {
	m, err := mgr.Connect()
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			return nil, errors.New("access denied — run this from an elevated (Administrator) prompt")
		}
		return nil, err
	}
	return m, nil
}

func ServiceInstall() error {
	cfg, cfgPath, err := config.InstallProgramDataConfig()
	if err != nil {
		return fmt.Errorf("prepare config: %w", err)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	m, err := connectManager()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	if s, err := m.OpenService(serviceName); err == nil {
		s.Close()
		return fmt.Errorf("service %q is already installed", serviceName)
	}
	s, err := m.CreateService(serviceName, exe, mgr.Config{
		DisplayName: serviceDisplay,
		Description: serviceDesc,
		StartType:   mgr.StartAutomatic,
		// ServiceStartName left empty => LocalSystem
	}, "service", "run")
	if err != nil {
		return fmt.Errorf("create service: %w", err)
	}
	s.Close()
	fmt.Printf("Installed service %q (LocalSystem, automatic start).\n", serviceName)
	fmt.Printf("Config: %s\n", cfgPath)
	fmt.Printf("Token:  %s\n", cfg.Token)
	fmt.Println("If this token differs from the widget's, paste it into the widget's Bridge Token setting.")
	fmt.Println("Start it now with:  xeneon-bridge service start")
	return nil
}

func ServiceUninstall(purge bool) error {
	m, err := connectManager()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %q is not installed", serviceName)
	}
	defer s.Close()
	// Best-effort stop first: Delete on a running service only marks it for
	// deletion, and a following --purge would race the still-running process
	// holding bridge.log open. Stopping first makes removal immediate.
	if st, err := s.Query(); err == nil && st.State != svc.Stopped {
		if _, err := s.Control(svc.Stop); err == nil {
			_ = waitState(s, svc.Stopped)
		}
	}
	if err := s.Delete(); err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	fmt.Printf("Removed service %q.\n", serviceName)
	if purge {
		dir := config.ProgramDataDir()
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("purge config: %w", err)
		}
		fmt.Printf("Purged config dir %s\n", dir)
	}
	return nil
}

func ServiceStart() error {
	m, err := connectManager()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %q is not installed", serviceName)
	}
	defer s.Close()
	if err := s.Start(); err != nil {
		return fmt.Errorf("start service: %w", err)
	}
	if err := waitState(s, svc.Running); err != nil {
		return err
	}
	fmt.Println("Service started.")
	return nil
}

func ServiceStop() error {
	m, err := connectManager()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(serviceName)
	if err != nil {
		return fmt.Errorf("service %q is not installed", serviceName)
	}
	defer s.Close()
	if _, err := s.Control(svc.Stop); err != nil {
		return fmt.Errorf("stop service: %w", err)
	}
	if err := waitState(s, svc.Stopped); err != nil {
		return err
	}
	fmt.Println("Service stopped.")
	return nil
}

func ServiceStatus() error {
	if p, err := config.Path(); err == nil {
		fmt.Printf("Config: %s\n", p)
	}
	m, err := mgr.Connect()
	if err != nil {
		if errors.Is(err, windows.ERROR_ACCESS_DENIED) {
			fmt.Println("Service state: run from an elevated prompt to query")
			return nil
		}
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(serviceName)
	if err != nil {
		fmt.Printf("Service %q: NotInstalled\n", serviceName)
		return nil
	}
	defer s.Close()
	st, err := s.Query()
	if err != nil {
		return err
	}
	fmt.Printf("Service %q: %s\n", serviceName, stateName(st.State))
	return nil
}

func waitState(s *mgr.Service, want svc.State) error {
	for i := 0; i < 20; i++ { // up to ~6s
		st, err := s.Query()
		if err != nil {
			return err
		}
		if st.State == want {
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for service state %s", stateName(want))
}

func stateName(s svc.State) string {
	switch s {
	case svc.Running:
		return "Running"
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "StartPending"
	case svc.StopPending:
		return "StopPending"
	default:
		return fmt.Sprintf("State(%d)", uint32(s))
	}
}
