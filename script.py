import os
import vapoursynth as vs
from vapoursynth import core
import vsmlrt


def process(src):
	rgb = core.resize.Bicubic(src, format=vs.RGBH, matrix_in_s="709")

	num_streams = int(os.getenv('VSPIPE_NUM_STREAMS', '1'))
	be = vsmlrt.Backend.TRT(fp16=True, tf32=False, output_format=1, use_cublas=False, use_cuda_graph=True,
							use_cudnn=False, num_streams=num_streams, force_fp16=True)

	model_path = os.getenv('VISPIPE_MODEL_PATH')
	if model_path is None:
		model_name = os.getenv('VSPIPE_MODEL_NAME', 'animejanaiV3_HD_L2')
		rgb = vsmlrt.RealESRGANv2(rgb, model=vsmlrt.RealESRGANv2Model[model_name], backend=be)
	else:
		rgb = vsmlrt.inference(rgb, model_path, backend=be)

	if os.getenv('VSPIPE_RIFE', '0') == '1':
		model_name = os.getenv('VSPIPE_RIFE_MODEL', 'v4_7')
		num_streams = int(os.getenv('VSPIPE_RIFE_NUM_STREAMS', '1'))
		be = vsmlrt.Backend.TRT(fp16=True, tf32=False, output_format=1, use_cublas=False, use_cuda_graph=True,
								use_cudnn=False, num_streams=num_streams)
		rgb = vsmlrt.RIFE(rgb, model=vsmlrt.RIFEModel[model_name].value,
						  ensemble=True, backend=be, scale=1.0, _implementation=1)

	video = core.resize.Bicubic(rgb, format=vs.YUV420P10, matrix_s="709")
	return video


args = globals()
src = core.bs.VideoSource(args['in'], cachemode=3, cachepath=args['cache'])
video = process(src[int(args['from']):int(args['to'])])
video.set_output()