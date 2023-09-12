FROM python:3.11.5
MAINTAINER AriesWang

RUN pip install --upgrade --no-deps --force-reinstall git+https://github.com/openai/whisper.git
RUN pip install torchvision tqdm tiktoken numba
RUN apt-get update
RUN apt-get upgrade
RUN apt-get install -y ffmpeg

