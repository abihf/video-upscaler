package upscaler

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/abihf/video-upscaler/internal/ffprog"
	"github.com/sirupsen/logrus"
)

type Task struct {
	Input   string
	Output  string
	TempDir string

	BackgroundFinalize bool

	log     *logrus.Logger
	logFile *os.File
}

func Demo(ctx context.Context) error {
	logrus.SetFormatter(&logrus.TextFormatter{ForceColors: true})
	t := Task{
		Input:   "/media/data/tmp/pendek.mkv",
		Output:  "/media/data/tmp/gede.mkv",
		TempDir: "/media/data/tmp/upscale",
	}
	return t.Upscale(ctx)
}

const FramesPerPart = 1000

func (t *Task) Upscale(ctx context.Context) error {
	t.log = logrus.New()

	if fileExists(t.Output) {
		t.log.WithField("out", t.Output).WithField("in", t.Input).Warn("Output already exist, skipping upscale")
		return nil
	}

	if !fileExists(t.Input) {
		return fmt.Errorf("input file %s not found", t.Input)
	}

	err := os.MkdirAll(t.TempDir, 0755)
	if err != nil {
		return fmt.Errorf("can not create temp dir %s: %w", t.TempDir, err)
	}

	logFileName := path.Join(t.TempDir, "upscale.log")
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("can not create progress file %s: %w", logFileName, err)
	}
	defer logFile.Close()
	defer logFile.WriteString("END\n--------------------------\n\n")
	t.logFile = logFile
	t.log.SetOutput(io.MultiWriter(t.log.Out, t.logFile))

	listFileName := path.Join(t.TempDir, "files.txt")
	err = t.upscaleParts(ctx, listFileName)
	if err != nil {
		return err
	}

	if t.BackgroundFinalize {
		go t.finalize(context.Background(), listFileName)
		return nil
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

	for fi := 0; fi < totalFrame; fi += FramesPerPart {
		partFileName := fmt.Sprintf("%s/%07d.mkv", t.TempDir, fi)
		fmt.Fprintf(listFile, "file %s\n", partFileName)
		if fileExists(partFileName) {
			continue
		}

		partFileTemp := fmt.Sprintf("%s/work-%07d.mkv", t.TempDir, fi)
		// progFileName := fmt.Sprintf("%s/%07d.ffprog", t.TempDir, fi)

		t.log.WithField("in", t.Input).WithField("frame", fi).Info("Upscaling part")
		err := t.upscalePart(ctx, fi, fi+FramesPerPart, partFileTemp)
		if err != nil {
			return err
		}

		t.log.WithField("path", partFileTemp).Info("Moving temporary part file")
		err = os.Rename(partFileTemp, partFileName)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *Task) upscalePart(ctx context.Context, from, to int, outfile string) error {
	lwi := path.Join(t.TempDir, "input.lwi")
	vspipe := exec.CommandContext(ctx, "vspipe",
		"-c", "y4m", "/upscale/script.vpy",
		"-a", "in="+t.Input,
		"-a", "lwi="+lwi,
		"-a", fmt.Sprintf("from=%d", from),
		"-a", fmt.Sprintf("to=%d", to),
		"-")
	vspipe.Stdin = os.Stdin
	vspipe.Stderr = os.Stderr

	ffmpeg := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error",
		"-i", "-",
		// ffv1
		// "-c:v", "ffv1", "-level", "3",
		//
		"-c:v", "hevc_nvenc", "-profile:v", "main10", "-preset:v", "slow", "-rc:v", "vbr", "-qmin:v", "24", "-qmax:v", "18",
		//

		"-y", outfile)
	ffmpeg.Stderr = os.Stderr
	ffmpeg.Stdin, _ = vspipe.StdoutPipe()
	ffmpeg.Stdout = os.Stdout
	ffprog.Handle(ffmpeg)

	errChan := make(chan error, 2)
	go runCmd(vspipe, errChan)
	go runCmd(ffmpeg, errChan)

	var err error
	for i := 0; i < 2; i++ {
		err = <-errChan
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *Task) finalize(ctx context.Context, listFileName string) error {
	combinedFile := path.Join(t.TempDir, "combined.mkv")
	t.log.WithField("target", combinedFile).Info("Combining files")
	ffmpeg := exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error",
		"-f", "concat", "-safe", "0", "-i", listFileName, "-f", "matroska", "-i", t.Input,
		"-map_metadata", "1", "-map", "0:v:0", "-map", "1", "-map", "-1:v:0", "-c:a", "copy",

		// "-c:v", "hevc_nvenc", "-profile:v", "main10", "-preset:v", "slow", "-rc:v", "vbr", "-qmin:v", "24", "-qmax:v", "20",
		"-c:v", "copy",
		"-y", combinedFile,
	)
	ffmpeg.Stdout = os.Stdout
	ffmpeg.Stderr = os.Stderr
	ffprog.Handle(ffmpeg)

	err := ffmpeg.Run()
	if err != nil {
		return err
	}

	t.log.WithField("temp", combinedFile).WithField("real", t.Output).Info("Moving combined files to output")
	err = os.Rename(combinedFile, t.Output)
	if err != nil {
		return err
	}

	t.log.WithField("path", t.TempDir).Info("Removing temporary video files")
	dirents, err := os.ReadDir(t.TempDir)
	if err != nil {
		return err
	}
	for _, dirent := range dirents {
		name := dirent.Name()
		if !dirent.IsDir() && strings.HasSuffix(".mkv", name) {
			os.Remove(path.Join(t.TempDir, name))
		}
	}

	return nil
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
	content, err := ioutil.ReadFile(frameCountFile)
	if err == nil {
		return content, nil
	}

	cmd := exec.Command("ffprobe", "-v", "error", "-select_streams", "v", "-count_packets", "-show_entries", "stream=nb_read_packets", "-of", "csv=p=0", t.Input)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	ioutil.WriteFile(frameCountFile, stdout, 0644)
	return stdout, nil
}
