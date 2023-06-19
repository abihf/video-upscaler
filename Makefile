docker:
	docker buildx build . -t abihf/video-upscaler --load \
		--cache-from type=local,src=docker-cache \
		--cache-to type=local,dest=docker-cache,mode=max
