package upscaler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/abihf/video-upscaler/internal/logstream"
)

var ffInputArgs = parseArgsFromEnv("FFMPEG_INPUT_ARGS", "-hide_banner", "-loglevel", "info", "-stats_period", "10")
var ffTranscodeArgs = parseArgsFromEnv("FFMPEG_TRANSCODE_ARGS", "-c:v", "hevc_nvenc", "-profile:v", "main10",
	"-preset:v", "slow", "-rc:v", "vbr", "-cq:v", "16", "-temporal_aq", "1", "-spatial_aq", "1", "-g", "24", "-strict_gop", "1")

type Task struct {
	Input  string
	Output string

	TempDir string
	log     *slog.Logger
	baseLog *slog.Logger
	// logFile *os.File
}

const FramesPerPart = 7200 // about 5 minutes for 24fps

func (t *Task) Upscale(ctx context.Context) error {
	if fileExists(t.Output) {
		slog.With("out", t.Output, "in", t.Input).Warn("Output already exist, skipping upscale")
		return nil
	}

	if !fileExists(t.Input) {
		return fmt.Errorf("input file %s not found", t.Input)
	}

	err := os.MkdirAll(t.TempDir, 0755)
	if err != nil {
		return fmt.Errorf("can not create temp dir %s: %w", t.TempDir, err)
	}

	slog.Info("Upscaling file", "in", t.Input, "out", t.Output, "tmp", t.TempDir)

	logFileName := path.Join(t.TempDir, "upscale.log")
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("can not create progress file %s: %w", logFileName, err)
	}
	defer logFile.Close()
	defer logFile.WriteString("\n -------------- CUT HERE -------------- \n\n")
	t.baseLog = slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{}))
	t.log = t.baseLog.With("app", "worker")

	listFileName := path.Join(t.TempDir, "files.txt")
	err = t.upscaleParts(ctx, listFileName)
	if err != nil {
		return err
	}

	err = t.finalize(ctx, listFileName)
	if err != nil {
		return err
	}

	return nil
}

func (t *Task) upscaleParts(ctx context.Context, listFileName string) error {
	totalFrame, err := t.getTotalFrame()
	if err != nil {
		return err
	}

	listFile, err := os.Create(listFileName)
	if err != nil {
		return fmt.Errorf("can not create list file %s: %w", listFileName, err)
	}
	defer listFile.Close()

	for frameIndex := 0; frameIndex < totalFrame; frameIndex += FramesPerPart {
		partFileName := fmt.Sprintf("%s/%07d+%d.mkv", t.TempDir, frameIndex, FramesPerPart)
		fmt.Fprintf(listFile, "file '%s'\n", partFileName)
		if fileExists(partFileName) {
			continue
		}

		partFileTemp := fmt.Sprintf("%s/work-%07d.mkv", t.TempDir, frameIndex)

		t.log.With("file", partFileTemp).Info("Upscaling part")
		err := t.upscalePart(ctx, frameIndex, frameIndex+FramesPerPart, partFileTemp)
		if err != nil {
			return err
		}

		t.log.With("path", partFileTemp).Info("Moving temporary part file")
		err = os.Rename(partFileTemp, partFileName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Task) upscalePart(ctx context.Context, from, to int, outfile string) error {
	cacheName := path.Join(t.TempDir, "cache")
	vspipe := exec.CommandContext(ctx, "vspipe",
		"-c", "y4m", "/upscale/script.py",
		"-a", "in="+t.Input,
		"-a", "cache="+cacheName,
		"-a", fmt.Sprintf("from=%d", from),
		"-a", fmt.Sprintf("to=%d", to),
		"-")
	vspipeOut, err := vspipe.StdoutPipe()
	if err != nil {
		return err
	}
	defer vspipeOut.Close()
	defer t.captureOutput(vspipe)()

	fullArgs := append(ffInputArgs, "-i", "-")
	fullArgs = append(fullArgs, ffTranscodeArgs...)
	fullArgs = append(fullArgs, "-y", outfile)
	ffmpeg := exec.CommandContext(ctx, "ffmpeg", fullArgs...)
	ffmpeg.Stdin = vspipeOut
	defer t.captureOutput(ffmpeg)()

	return awaitAll(func(cmd *exec.Cmd) error {
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("%s error: %w", cmd.Path, err)
		}
		return nil
	}, vspipe, ffmpeg)
}

func (t *Task) finalize(ctx context.Context, listFileName string) error {
	combinedFile := path.Join(t.TempDir, "combined.mkv")
	t.log.With("target", combinedFile).Info("Combining files")
	// combine the video files and merge it with original audio & subtitles
	ffmpeg := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "info",
		"-f", "concat", "-safe", "0", "-i", listFileName, "-f", "matroska", "-i", t.Input,
		"-map_metadata", "1", "-map", "0:v:0", "-map", "1", "-map", "-1:v:0", "-c", "copy", "-seek2any", "1",
		"-g", "24",
		"-y", combinedFile,
	)
	defer t.captureOutput(ffmpeg)()

	err := ffmpeg.Run()
	if err != nil {
		return err
	}

	t.log.Info("Moving combined files to output", "temp", combinedFile, "real", t.Output)
	err = t.moveFile(combinedFile, t.Output)
	if err != nil {
		return err
	}

	t.log.Info("Removing temporary video files", "path", t.TempDir)
	dirents, err := os.ReadDir(t.TempDir)
	if err != nil {
		return err
	}
	for _, dirent := range dirents {
		name := dirent.Name()
		if !dirent.IsDir() && strings.HasSuffix(name, ".mkv") {
			os.Remove(path.Join(t.TempDir, name))
		}
	}

	return nil
}

func (t *Task) moveFile(src, dest string) error {
	err := os.Link(src, dest)
	if err != nil {
		t.log.Warn("can not create hard link", "err", err, "src", src, "dest", dest)
		return os.Rename(src, dest)
	}
	return os.Remove(src)
}

func (t *Task) getTotalFrame() (int, error) {
	str, err := t.getTotalFrameStr()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(str)))
}

func (t *Task) getTotalFrameStr() ([]byte, error) {
	frameCountFile := path.Join(t.TempDir, "framecount")
	content, err := os.ReadFile(frameCountFile)
	if err == nil {
		return content, nil
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v", "-count_packets", "-show_entries", "stream=nb_read_packets", "-of", "csv=p=0", t.Input)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	lnIndex := bytes.IndexByte(stdout, '\n')
	os.WriteFile(frameCountFile, stdout[:lnIndex], 0644)
	return stdout, nil
}

func (t *Task) captureOutput(cmd *exec.Cmd) func() {
	var stdout, stderr io.WriteCloser
	if t.baseLog != nil {
		appLogger := t.baseLog.With("app", path.Base(cmd.Path))
		appLogger.Info("Run process", "args", cmd.Args)
		if cmd.Stdout == nil {
			stdout = logstream.New(func(line string) error {
				appLogger.Info(line)
				return nil
			})
			cmd.Stdout = stdout
		}
		if cmd.Stderr == nil {
			stderr = logstream.New(func(line string) error {
				appLogger.Warn(line)
				return nil
			})
			cmd.Stderr = stderr
		}
	}

	return func() {
		if stdout != nil {
			stdout.Close()
		}
		if stderr != nil {
			stderr.Close()
		}
	}
}
