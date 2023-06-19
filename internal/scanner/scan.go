package scanner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
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
	files, err := s.scanSubDir(ctx, "", false)
	if err != nil {
		return err
	}
	sort.Strings(files)

	for _, file := range files {
		out := getUhdName(file)
		payload, _ := json.Marshal(model.VideoUpscaleTask{
			In:  file,
			Out: out,
		})
		logrus.WithField("in", file).Info("Add to queue")
		_, err := s.AsynqClient.EnqueueContext(ctx, asynq.NewTask(model.TaskVideoUpscaleType, payload,
			asynq.TaskID(out),
			asynq.Timeout(3*time.Hour),
			asynq.MaxRetry(2),
		))
		if err != nil {
			logrus.WithField("in", file).WithError(err).Error("Can not add to queue")
		}
	}

	return nil
}

func (s *Scanner) scanSubDir(ctx context.Context, subDir string, active bool) ([]string, error) {
	fullPath := path.Join(s.Root, subDir)
	dirents, err := readDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("can't list dir: %w", err)
	}

	if _, ok := dirents[".upscale"]; ok {
		logrus.WithField("subdir", subDir).Info("Marker file .upscale found")
		active = true
	}

	upscaleFiles := []string{}
	hdFiles := map[string]string{}
	uhdFilesExist := map[string]bool{}
	for name, dirent := range dirents {
		if dirent.IsDir() {
			currentRelPath := path.Join(subDir, name)
			subdirFiles, err := s.scanSubDir(ctx, currentRelPath, active)
			if err != nil {
				logrus.WithContext(ctx).WithError(err).WithField("path", currentRelPath).Error("Can not process subdir")
			} else if len(subdirFiles) > 0 {
				upscaleFiles = append(upscaleFiles, subdirFiles...)
			}
			continue
		}
		if !active {
			continue
		}
		if strings.HasSuffix(".mkv", name) {
			// only support mkv file
			continue
		}

		if isHdFile(name) {
			se := getSeasonEpisode(name)
			if se != "" {
				hdFiles[se] = name
			}
		} else if isUhdFile(name) {
			se := getSeasonEpisode(name)
			if se != "" {
				uhdFilesExist[se] = true
			}
		}
	}

	for se, name := range hdFiles {
		if uhdFilesExist[se] {
			continue
		}
		upscaleFiles = append(upscaleFiles, path.Join(s.Root, subDir, name))
	}

	return upscaleFiles, nil
}

func isHdFile(file string) bool {
	return strings.Contains(file, "1080p")
}

func isUhdFile(file string) bool {
	return strings.Contains(file, "2160p") || strings.Contains(file, "-4k")
}

func getUhdName(file string) string {
	idx := strings.LastIndexByte(file, '/') + 1
	return file[:idx] + strings.ReplaceAll(file[idx:], "1080p", "2060p")
}

var reEpisode = regexp.MustCompile(`(?i)S\d+E\d+(-\d+)?`)

func getSeasonEpisode(file string) string {
	return reEpisode.FindString(file)
}

func readDir(dir string) (map[string]os.DirEntry, error) {
	contents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	mapped := make(map[string]os.DirEntry, len(contents))
	for _, content := range contents {
		mapped[content.Name()] = content
	}
	return mapped, nil
}
