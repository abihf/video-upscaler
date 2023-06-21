package scanner

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/abihf/video-upscaler/internal/model"
	"github.com/hibiken/asynq"
	"github.com/sirupsen/logrus"
)

type Scanner struct {
	Root        string
	AsynqClient *asynq.Client
}

func (s *Scanner) Scan(ctx context.Context) error {
	startTime := time.Now()
	err := s.scanSubDir(ctx, "", false)
	if err != nil {
		return err
	}
	logrus.WithField("duration", time.Since(startTime)).Info("Done scanning")

	return nil
}

func (s *Scanner) scanSubDir(ctx context.Context, subDir string, active bool) error {
	fullPath := path.Join(s.Root, subDir)
	dirents, err := os.ReadDir(fullPath)
	if err != nil {
		return fmt.Errorf("can't list dir: %w", err)
	}

	if !active && hasMarkerFile(dirents) {
		logrus.WithField("subdir", subDir).Info("Marker file .upscale found")
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
					logrus.WithContext(ctx).WithError(err).WithField("path", relPath).Error("Can not process subdir")
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
	payload, _ := json.Marshal(model.VideoUpscaleTask{
		In:  file,
		Out: out,
	})

	queueName := "default"
	if time.Since(stat.ModTime()) <= 6*time.Hour {
		queueName = "critical"
	}

	id := sha1.Sum([]byte(out))
	task := asynq.NewTask(model.TaskVideoUpscaleType, payload,
		asynq.Timeout(3*time.Hour),
		asynq.MaxRetry(2),
		asynq.Retention(30*24*time.Hour),
		asynq.TaskID(base64.RawURLEncoding.EncodeToString(id[:])),
		asynq.Queue(queueName),
	)

	_, err = s.AsynqClient.EnqueueContext(ctx, task)
	if err != nil {
		if errors.Is(err, asynq.ErrTaskIDConflict) || errors.Is(err, asynq.ErrDuplicateTask) {
			logrus.WithField("in", file).WithError(err).Debug("Already in queue")
		} else {
			return err
		}
	} else {
		logrus.WithField("in", file).Info("Added to queue")
	}
	return nil
}

func hasMarkerFile(list []os.DirEntry) bool {
	for _, de := range list {
		if de.Name() == ".upscale" {
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
