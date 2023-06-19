import os
import vapoursynth as vs
from vapoursynth import core
import vsmlrt


def process(src):
    # adjust 32 and 31 to match specific AI network input resolution requirements.
    th = (src.height + 15) // 16 * 16
    tw = (src.width + 15) // 16 * 16  # same.
    rgb = core.resize.Bicubic(
        src, tw, th, format=vs.RGBH, matrix_in_s="709", src_width=tw, src_height=th)

    be = vsmlrt.Backend.TRT(fp16=True, tf32=False, output_format=1, use_cublas=True, use_cuda_graph=True,
                            use_cudnn=False, num_streams=1, force_fp16=True)

    model = vsmlrt.RealESRGANv2Model.animejanaiL2_std
    rgb = vsmlrt.RealESRGANv2(rgb, model=model, backend=be)

    if os.getenv("RIFE", "0") == "1":
        rgb = vsmlrt.RIFE(rgb, model=vsmlrt.RIFEModel.v4_6,
                          ensemble=True, backend=be, scale=1.0, _implementation=1)

    # not necessary for RIFE (i.e. oh = src.height), but required for super-resolution upscalers.
    oh = src.height * (rgb.height // th)
    ow = src.width * (rgb.width // tw)
    video = core.resize.Bicubic(
        rgb, ow, oh, format=vs.YUV420P10, matrix_s="709", src_width=ow, src_height=oh)
    return video


args = globals()
src = core.lsmas.LWLibavSource(args['in'], prefer_hw=1, cachefile=args['lwi'])
video = process(src[int(args['from']):int(args['to'])])
video.set_output()
