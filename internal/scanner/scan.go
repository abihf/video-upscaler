package scanner

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/abihf/video-upscaler/internal/queue"
	"github.com/hibiken/asynq"
)

const MarkerFileName = ".upscale"

type Scanner struct {
	Root        string
	AsynqClient *asynq.Client
}

func (s *Scanner) Scan(ctx context.Context) error {
	startTime := time.Now()
	active := s.parentHasMarker()
	err := s.scanSubDir(ctx, "", active)
	if err != nil {
		return err
	}
	slog.With("duration", time.Since(startTime)).Info("Done scanning")

	return nil
}

func (s *Scanner) parentHasMarker() bool {
	dir := s.Root
	for {
		// slog.Info("Checking", "path", fullPath)
		if fileExists(path.Join(dir, MarkerFileName)) {
			return true
		}
		parent := path.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return false
}

func (s *Scanner) scanSubDir(ctx context.Context, subDir string, active bool) error {
	fullPath := path.Join(s.Root, subDir)

	dirents, err := os.ReadDir(fullPath)
	if err != nil {
		return fmt.Errorf("can't list dir: %w", err)
	}

	if !active && hasMarkerFile(dirents) {
		slog.With("subdir", subDir).Info("Marker file .upscale found")
		active = true
	}

	hdFiles := map[string]string{}
	uhdFilesExist := map[string]bool{}
	var wg sync.WaitGroup
	for _, dirent := range dirents {
		name := dirent.Name()
		if name[0] == '.' {
			// ignore dot files
			continue
		}

		if dirent.IsDir() {
			wg.Add(1)
			go func(relPath string) {
				defer wg.Done()
				err := s.scanSubDir(ctx, relPath, active)
				if err != nil {
					slog.Error("Can not process subdir", "err", err, "path", relPath)
				}
			}(path.Join(subDir, name))

			continue
		}
		if !active {
			continue
		}
		if !strings.HasSuffix(name, ".mkv") {
			// only support mkv file
			continue
		}

		// group file by season and episode
		se := getSeasonEpisode(name)
		if se != "" {
			if isHdFile(name) {
				hdFiles[se] = name
			} else if isUhdFile(name) {
				uhdFilesExist[se] = true
			}
		}
	}

	upscaleFile := []string{}
	for se, name := range hdFiles {
		if !uhdFilesExist[se] {
			upscaleFile = append(upscaleFile, name)
		}
	}

	// sort the list so earlier episode get upscaled first
	sort.Strings(upscaleFile)

	for _, name := range upscaleFile {
		err := s.processFile(ctx, path.Join(s.Root, subDir, name))
		if err != nil {
			return err
		}
	}

	wg.Wait()
	return nil
}

func (s *Scanner) processFile(ctx context.Context, file string) error {
	stat, err := os.Stat(file)
	if err != nil {
		return err
	}
	out := getUhdName(file)

	priority := "default"
	if time.Since(stat.ModTime()) <= 6*time.Hour {
		priority = "critical"
	}

	err = queue.Add(ctx, s.AsynqClient, file, out, priority)
	if err != nil {
		if errors.Is(err, asynq.ErrTaskIDConflict) || errors.Is(err, asynq.ErrDuplicateTask) {
			slog.With("in", file, "err", err).Debug("Already in queue")
		} else {
			return err
		}
	} else {
		slog.With("in", file).Info("Added to queue")
	}
	return nil
}

func hasMarkerFile(list []os.DirEntry) bool {
	for _, de := range list {
		if de.Name() == MarkerFileName {
			return true
		}
	}
	return false
}

func isHdFile(file string) bool {
	return strings.Contains(file, "1080p")
}

func isUhdFile(file string) bool {
	return strings.Contains(file, "2160p") || strings.Contains(file, "-4k")
}

func getUhdName(file string) string {
	idx := strings.LastIndexByte(file, '/') + 1
	return file[:idx] + strings.ReplaceAll(file[idx:], "1080p", "2160p")
}

var reEpisode = regexp.MustCompile(`(?i)S\d+E\d+(-\d+)?`)

func getSeasonEpisode(file string) string {
	return reEpisode.FindString(file)
}
