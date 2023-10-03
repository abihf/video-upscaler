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

	num_streams = int(os.getenv('VSPIPE_NUM_STREAMS', '1'))
	be = vsmlrt.Backend.TRT(fp16=True, tf32=False, output_format=1, use_cublas=True, use_cuda_graph=True,
							use_cudnn=False, num_streams=num_streams, force_fp16=True)

	model_path = os.getenv('VISPIPE_MODEL_PATH')
	if model_path is None:
		model_name = os.getenv('VSPIPE_MODEL_NAME', 'animejanaiV2L2')
		rgb = vsmlrt.RealESRGANv2(rgb, model=vsmlrt.RealESRGANv2Model[model_name], backend=be)
	else:
		rgb = vsmlrt.inference(rgb, model_path, backend=be)

	if os.getenv('VSPIPE_RIFE', '0') == '1':
		model_name = os.getenv('VSPIPE_RIFE_MODEL', 'v4_7')
		num_streams = int(os.getenv('VSPIPE_RIFE_NUM_STREAMS', '1'))
		be = vsmlrt.Backend.TRT(fp16=True, tf32=False, output_format=1, use_cublas=True, use_cuda_graph=True,
								use_cudnn=False, num_streams=num_streams, force_fp16=True)
		rgb = vsmlrt.RIFE(rgb, model=vsmlrt.RIFEModel[model_name].value,
						  ensemble=True, backend=be, scale=1.0, _implementation=1)

	# not necessary for RIFE (i.e. oh = src.height), but required for super-resolution upscalers.
	oh = src.height * (rgb.height // th)
	ow = src.width * (rgb.width // tw)
	video = core.resize.Bicubic(
		rgb, ow, oh, format=vs.YUV420P10, matrix_s="709", src_width=ow, src_height=oh)
	return video


args = globals()
src = core.lsmas.LWLibavSource(args['in'], prefer_hw=0, cachefile=args['lwi'])
video = process(src[int(args['from']):int(args['to'])])
video.set_output()
