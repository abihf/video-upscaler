package model

const (
	TaskVideoUpscaleType = "video:upscale"
)

type VideoUpscaleTask struct {
	In  string
	Out string
	// TempFile string
}
