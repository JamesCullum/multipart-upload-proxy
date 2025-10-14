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


## Resize Strategies

The proxy supports two different image resizing strategies:

### Bounding Box Strategy (Default)
When `IMG_MAX_NARROW_SIDE` is not set (or set to 0), the proxy uses an intelligent orientation-aware bounding box approach:

**Orientation-Aware Resizing:**
- **Landscape images** (width ≥ height): Use `IMG_MAX_WIDTH` × `IMG_MAX_HEIGHT` limits directly
- **Portrait images** (height > width): Automatically swap to `IMG_MAX_HEIGHT` × `IMG_MAX_WIDTH` limits

**Examples with default 1920×1080 configuration:**
- 3000×2000 landscape → 1620×1080 (fits within 1920×1080 box)
- 2000×3000 portrait → 1080×1620 (fits within 1080×1920 swapped box)

This ensures optimal sizing for both orientations without manual configuration.

### Narrow Side Strategy
When `IMG_MAX_NARROW_SIDE` is set to a value greater than 0, the proxy constrains only the narrow side (shorter dimension) of the image to the specified limit. The wide side can be larger, which is useful for panoramic images or when you want to preserve more detail in one dimension.

For example, with `IMG_MAX_NARROW_SIDE=600`:
- A 2000x800 landscape image becomes 1500x600 (narrow side constrained to 600)
- A 800x2000 portrait image becomes 600x1500 (narrow side constrained to 600)

### Format Conversion

The proxy can convert images to different formats for optimization:

#### Format Selection
- **Disabled** (`CONVERT_TO_FORMAT=""`, default): Resize images without format conversion (backward compatible)
- **JPEG** (`CONVERT_TO_FORMAT="JPEG"` or `"JPG"` or `"jpeg"` or `"jpg"`): Convert images to JPEG format
- **WebP** (`CONVERT_TO_FORMAT="WEBP"` or `"webp"`): Convert images to WebP format

#### Transparency Handling
When converting images with transparency:
- **Any format conversion**: Transparent images are skipped entirely to preserve transparency
- **Rationale**: JPEG doesn't support transparency, and WebP loses alpha channel with current libvips/bimg setup

#### File Extension Normalization
For successfully converted images, you can choose whether to normalize file extensions:

- **Enabled** (`NORMALIZE_EXTENSIONS=1`, default): When conversion happens, `photo.png` → `photo.JPG` or `photo.WEBP`
- **Disabled** (`NORMALIZE_EXTENSIONS=0`): Keep original filenames even for converted images

When conversion doesn't reduce file size, the original image format and filename are preserved to maintain quality.

## Environment variables

|Variable name                          |Default                         | Comment
|-------------------------------|-----------------------------| -----------------------------|
|`IMG_MAX_WIDTH`            |1920            | Pixels, keeps aspect ratio (ignored if IMG_MAX_NARROW_SIDE is set)
|`IMG_MAX_HEIGHT`            |1080            | Pixels, keeps aspect ratio (ignored if IMG_MAX_NARROW_SIDE is set)
|`IMG_MAX_NARROW_SIDE`      |0 (disabled)    | Pixels, constrains the narrow side of the image, allows wide side to be larger
|`JPEG_QUALITY`|90|JPEG compression quality (1-100, lower = smaller file). Invalid values fall back to default
|`WEBP_QUALITY`|90|WebP compression quality (1-100, lower = smaller file). Invalid values fall back to default
|`CONVERT_TO_FORMAT`|"" (disabled)|Format conversion: "" (disabled), "JPEG"/"JPG"/"jpeg"/"jpg", or "WEBP"/"webp". Transparent images may fallback to PNG
|`NORMALIZE_EXTENSIONS`|1 (enabled)|Normalize filenames to converted format extension (1=enabled, 0=keep original names). Invalid values fall back to default
|`UPLOAD_MAX_SIZE`|104857600|Maximum form size in bytes
|`IMG_MAX_PIXELS`|2073600|If the images width*height (in pixels) doesn't exceed this value, don't resize. Defaults to IMG_MAX_WIDTH × IMG_MAX_HEIGHT
|`FORWARD_DESTINATION`|https://httpbin.org/anything|Where should the result be sent to
|`FILE_UPLOAD_FIELD`|assetData|Name of the file field to potentially resize
|`LISTEN_PATH`|/api/assets|Path used to process file uploads

