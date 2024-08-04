# Multipart Upload Proxy (Golang)

There are many tools to help you resize images when fetching resources from your online storage. However, sometimes you want to resize large images during an upload automatically instead. Especially if you don't have control over the software that is supposed to process the uploaded image, for example because it's open source and the [contributors don't think resizing should be a feature](https://github.com/immich-app/immich/pull/1242), getting the feature into the existing code base can be difficult. 

This is where the multipart upload proxy comes into play. You can route all multipart file uploads to the proxy and it will digest and resize images to the size you want, finally relaying the same payload with all headers and just a compressed file to the endpoint that saves the file.

The proxy is written in Golang and packaged in a small and safe Alpine container. If you want to develop, run or compile the binary, please be aware that the image resizing uses the [bimg](https://github.com/h2non/bimg) library, which requires a linux vips environment. If you're in Windows, usage of WSL is highly recommended.

## Usage
Use the docker container like below. It uses the [Github docker registry with signed docker containers](https://github.com/JamesCullum/multipart-upload-proxy/pkgs/container/multipart-upload-proxy). Use environmental variables to adjust settings. The container exposes port 6743 via HTTP.

    docker run --rm -p 6743:6743 --name=multipart-upload-proxy ghcr.io/jamescullum/multipart-upload-proxy:main

In docker compose, you can use it like this (if you only want it to be exposed within the docker network).

      upload-proxy:
        container_name: upload_proxy
        image: ghcr.io/jamescullum/multipart-upload-proxy:main
        environment:
          - IMG_MAX_WIDTH=1920
          - IMG_MAX_HEIGHT=1080
          - FORWARD_DESTINATION=http://immich-server:3001/api/assets
          - FILE_UPLOAD_FIELD=assetData
          - LISTEN_PATH=/api/assets
        restart: always

If you use existing software, it might be needed to intercept incoming connections and redirect them to this proxy. You can do this via Cloudflare tunnels or via a front-facing reverse proxy/webserver.


## Environment variables

|Variable name                          |Default                         | Comment
|-------------------------------|-----------------------------| -----------------------------| 
|`IMG_MAX_WIDTH`            |1920            | Pixels, keeps aspect ratio
|`IMG_MAX_HEIGHT`            |1080            | Pixels, keeps aspect ratio
|`UPLOAD_MAX_SIZE`|104857600|Maximum form size in bytes
|`IMG_MAX_PIXELS`|2073600|If the images width*height (in pixels) doesn't exceed this value, don't resize
|`FORWARD_DESTINATION`|https://httpbin.org/post|Where should the result be sent to
|`FILE_UPLOAD_FIELD`|assetData|Name of the file field to potentially resize
|`LISTEN_PATH`|/api/assets|Path used to process file uploads

