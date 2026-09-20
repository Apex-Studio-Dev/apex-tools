FROM ubuntu:24.04

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    bash ca-certificates curl git jq xz-utils zip unzip p7zip-full tar gzip \
    golang-go file \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /work
ENV PATH=/work/scripts:/work/cmd/apex:${PATH}
