

 
build_img:
		docker rmi -f arieswangdocker/yt2text:latest
		docker buildx create --use
		docker buildx build -t arieswangdocker/yt2text:latest --platform linux/amd64,linux/arm64 --push .

build_web:
		cd flutter && flutter build web

build_server:
		cd flutter && flutter build web

copy_web:
		rm -rf server/web
		cp -r flutter/build/web server/web