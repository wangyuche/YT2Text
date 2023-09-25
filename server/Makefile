
build_img_mac:
		export CGO_ENABLED=0 && \
		export GOOS=linux && \
		export GOARCH=arm64 && \
		go build -o app
		docker rmi -f arieswangdocker/yt2text:latest
		docker buildx create --use
		docker buildx build -t arieswangdocker/yt2text:latest --platform linux/arm64 --push .

build_img_linux:
		export CGO_ENABLED=0 && \
		export GOOS=linux && \
		export GOARCH=amd64 && \
		go build -o app
		docker rmi -f arieswangdocker/yt2text:latest
		docker buildx create --use
		docker buildx build -t arieswangdocker/yt2text:latest --platform linux/amd64 --push .
