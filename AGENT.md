# AGENT.md

Guidance for future AI/code agents working in this repository.

## Project summary

- `video-upscaler` is a Go CLI + Temporal worker for anime/video upscaling (1080p ➜ 2160p).
- Queue commands:
  - `video-upscaler add ...` for single files
  - `video-upscaler scan ...` for recursive library enqueue
- Worker command:
  - `video-upscaler worker`

## Processing workflow

The `Upscale` Temporal workflow runs activities in this order:

1. `Prepare`
2. `Info`
3. `Upscale`
4. `Merge`
5. `MoveFile`
6. `Delete`

Queue and namespace are both `upscaler`.

## VapourSynth script

`/script.py` is used by `vspipe` in the `Upscale` activity:

1. Load source via BestSource (`core.bs.VideoSource`)
2. Convert to `RGBH` (Rec.709)
3. Upscale with `vsmlrt` (default `RealESRGANv2`)
4. Optionally run RIFE interpolation
5. Convert to `YUV420P10` (Rec.709) and output

Important env var note:

- `script.py` currently reads `VISPIPE_MODEL_PATH` (known naming quirk: nonstandard `VISPIPE` spelling in current code; docs should match runtime behavior).

## Build/test

From repository root:

```bash
go test ./...
go build ./...
```

## Base image stack

`/base-image/Dockerfile` builds runtime dependencies including:

- Ubuntu 24.04 base + clang/lld 22 toolchain
- VapourSynth R73 + BestSource R16
- FFmpeg n8.0.1 (NVENC/NVDEC enabled) + mkvtoolnix
- vs-mlrt + CUDA 13.1 + TensorRT RTX 1.3.0.35
